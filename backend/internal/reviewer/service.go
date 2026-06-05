package reviewer

import (
	"context"
	"fmt"
	"log"
	"time"

	"tax-ocr/backend/internal/db"
)

const defaultExpireMinutes = 30

// reviewerTypeForTask maps task_type → reviewer_type in the reviewers table.
var reviewerTypeForTask = map[string]string{
	"text_verification":          "text_verifier",
	"classification_verification": "classification_verifier",
}

// taskTypesForReason maps HITL reason → one or more task types to create.
func taskTypesForReason(reason string) []string {
	switch reason {
	case "ocr_mismatch":
		return []string{"text_verification"}
	case "classification_needed":
		return []string{"classification_verification"}
	default:
		return []string{"text_verification", "classification_verification"}
	}
}

type Service struct {
	store         *db.Store
	line          *LineClient
	expireMinutes int
}

func NewService(store *db.Store, line *LineClient, expireMinutes int) *Service {
	if expireMinutes <= 0 {
		expireMinutes = defaultExpireMinutes
	}
	return &Service{store: store, line: line, expireMinutes: expireMinutes}
}

// AssignForHitl assigns reviewer task(s) for a HITL item using round-robin.
func (s *Service) AssignForHitl(ctx context.Context, hitl db.HitlQueueItem) {
	rewardMap, _ := s.store.GetRewardAmounts(ctx)

	for _, taskType := range taskTypesForReason(hitl.Reason) {
		rType, ok := reviewerTypeForTask[taskType]
		if !ok {
			continue
		}
		reviewer, err := s.store.NextReviewerRoundRobin(ctx, rType)
		if err != nil {
			log.Printf("[reviewer] no active %s available for hitl %s: %v", rType, hitl.ID, err)
			continue
		}

		expiredAt := time.Now().Add(time.Duration(s.expireMinutes) * time.Minute)
		task, err := s.store.CreateReviewerTaskWithExpiry(ctx, db.ReviewerTask{
			HitlQueueID:  hitl.ID,
			ReviewerID:   reviewer.ID,
			TaskType:     taskType,
			RewardAmount: rewardMap[taskType],
		}, expiredAt)
		if err != nil {
			log.Printf("[reviewer] create task error for hitl %s: %v", hitl.ID, err)
			continue
		}

		msg := fmt.Sprintf("งานใหม่มาถึง! (%s)\nรายการ ID: %s\nกรุณาตอบรับภายใน %d นาที",
			taskType, hitl.InvoiceItemID, s.expireMinutes)
		if err := s.line.Push(reviewer.LineUserID, msg); err != nil {
			log.Printf("[reviewer] line push error for reviewer %s: %v", reviewer.Name, err)
		}
		log.Printf("[reviewer] assigned task %s → reviewer %s (%s)", task.ID, reviewer.Name, taskType)
	}
}

// RunExpiryChecker starts a background goroutine that expires overdue tasks and reassigns them.
func (s *Service) RunExpiryChecker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.expireAndReassign(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) expireAndReassign(ctx context.Context) {
	expired, err := s.store.ExpireOldTasks(ctx)
	if err != nil {
		log.Printf("[reviewer] expire tasks error: %v", err)
		return
	}
	if len(expired) == 0 {
		return
	}
	log.Printf("[reviewer] expired %d task(s), reassigning", len(expired))
	for _, task := range expired {
		hitl, err := s.store.GetHitlItem(ctx, task.HitlQueueID)
		if err != nil {
			log.Printf("[reviewer] get hitl %s error: %v", task.HitlQueueID, err)
			continue
		}
		if hitl.Status != "pending" {
			continue
		}
		s.AssignForHitl(ctx, hitl)
	}
}
