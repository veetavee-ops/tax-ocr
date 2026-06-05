package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) ListReviewers(ctx context.Context) ([]Reviewer, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, line_user_id, reviewer_type, status, total_earned, pending_payout, created_at, updated_at
		 FROM reviewers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Reviewer
	for rows.Next() {
		var r Reviewer
		if err := rows.Scan(&r.ID, &r.Name, &r.LineUserID, &r.ReviewerType, &r.Status,
			&r.TotalEarned, &r.PendingPayout, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, nil
}

func (s *Store) GetReviewer(ctx context.Context, id string) (Reviewer, error) {
	var r Reviewer
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, line_user_id, reviewer_type, status, total_earned, pending_payout, created_at, updated_at
		 FROM reviewers WHERE id = $1`, id).
		Scan(&r.ID, &r.Name, &r.LineUserID, &r.ReviewerType, &r.Status,
			&r.TotalEarned, &r.PendingPayout, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reviewer{}, ErrNotFound
	}
	return r, err
}

func (s *Store) CreateReviewer(ctx context.Context, input Reviewer) (Reviewer, error) {
	if input.Name == "" || input.LineUserID == "" || input.ReviewerType == "" {
		return Reviewer{}, ErrInvalidInput
	}
	var r Reviewer
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reviewers (name, line_user_id, reviewer_type)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, line_user_id, reviewer_type, status, total_earned, pending_payout, created_at, updated_at`,
		input.Name, input.LineUserID, input.ReviewerType).
		Scan(&r.ID, &r.Name, &r.LineUserID, &r.ReviewerType, &r.Status,
			&r.TotalEarned, &r.PendingPayout, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Reviewer{}, ErrDuplicateLineUserID
		}
		return Reviewer{}, err
	}
	return r, nil
}

func (s *Store) UpdateReviewer(ctx context.Context, id string, input Reviewer) (Reviewer, error) {
	var r Reviewer
	err := s.pool.QueryRow(ctx,
		`UPDATE reviewers SET
			name          = COALESCE(NULLIF($2,''), name),
			reviewer_type = COALESCE(NULLIF($3,''), reviewer_type),
			status        = COALESCE(NULLIF($4,''), status),
			updated_at    = NOW()
		 WHERE id = $1
		 RETURNING id, name, line_user_id, reviewer_type, status, total_earned, pending_payout, created_at, updated_at`,
		id, input.Name, input.ReviewerType, input.Status).
		Scan(&r.ID, &r.Name, &r.LineUserID, &r.ReviewerType, &r.Status,
			&r.TotalEarned, &r.PendingPayout, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reviewer{}, ErrNotFound
	}
	return r, err
}

func (s *Store) ListReviewerTasks(ctx context.Context, reviewerID string) ([]ReviewerTask, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		  sent_at, accepted_at, completed_at, expired_at, created_at, updated_at
		 FROM reviewer_tasks WHERE reviewer_id = $1 ORDER BY created_at DESC`, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ReviewerTask
	for rows.Next() {
		var t ReviewerTask
		if err := rows.Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, nil
}

func (s *Store) CreateReviewerTask(ctx context.Context, input ReviewerTask) (ReviewerTask, error) {
	if input.HitlQueueID == "" || input.ReviewerID == "" || input.TaskType == "" {
		return ReviewerTask{}, ErrInvalidInput
	}
	var t ReviewerTask
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reviewer_tasks (hitl_queue_id, reviewer_id, task_type, reward_amount, sent_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		   sent_at, accepted_at, completed_at, expired_at, created_at, updated_at`,
		input.HitlQueueID, input.ReviewerID, input.TaskType, input.RewardAmount).
		Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *Store) CreateReviewerTaskWithExpiry(ctx context.Context, input ReviewerTask, expiredAt time.Time) (ReviewerTask, error) {
	if input.HitlQueueID == "" || input.ReviewerID == "" || input.TaskType == "" {
		return ReviewerTask{}, ErrInvalidInput
	}
	var t ReviewerTask
	err := s.pool.QueryRow(ctx,
		`INSERT INTO reviewer_tasks (hitl_queue_id, reviewer_id, task_type, reward_amount, sent_at, expired_at)
		 VALUES ($1, $2, $3, $4, NOW(), $5)
		 RETURNING id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		   sent_at, accepted_at, completed_at, expired_at, created_at, updated_at`,
		input.HitlQueueID, input.ReviewerID, input.TaskType, input.RewardAmount, expiredAt).
		Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *Store) GetReviewerTask(ctx context.Context, id string) (ReviewerTask, error) {
	var t ReviewerTask
	err := s.pool.QueryRow(ctx,
		`SELECT id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		  sent_at, accepted_at, completed_at, expired_at, created_at, updated_at
		 FROM reviewer_tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewerTask{}, ErrNotFound
	}
	return t, err
}

// NextReviewerRoundRobin returns the active reviewer of given type with the oldest last assignment.
func (s *Store) NextReviewerRoundRobin(ctx context.Context, reviewerType string) (Reviewer, error) {
	var r Reviewer
	err := s.pool.QueryRow(ctx,
		`SELECT r.id, r.name, r.line_user_id, r.reviewer_type, r.status, r.total_earned, r.pending_payout, r.created_at, r.updated_at
		 FROM reviewers r
		 WHERE r.reviewer_type = $1 AND r.status = 'active'
		 ORDER BY (
		   SELECT COALESCE(MAX(t.sent_at), '1970-01-01'::timestamptz)
		   FROM reviewer_tasks t WHERE t.reviewer_id = r.id
		 ) ASC
		 LIMIT 1`, reviewerType).
		Scan(&r.ID, &r.Name, &r.LineUserID, &r.ReviewerType, &r.Status,
			&r.TotalEarned, &r.PendingPayout, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reviewer{}, ErrNotFound
	}
	return r, err
}

// ExpireOldTasks marks overdue 'sent' tasks as 'expired' and returns the list.
func (s *Store) ExpireOldTasks(ctx context.Context) ([]ReviewerTask, error) {
	rows, err := s.pool.Query(ctx,
		`UPDATE reviewer_tasks SET status = 'expired', updated_at = NOW()
		 WHERE status = 'sent' AND expired_at IS NOT NULL AND expired_at < NOW()
		 RETURNING id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		   sent_at, accepted_at, completed_at, expired_at, created_at, updated_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ReviewerTask
	for rows.Next() {
		var t ReviewerTask
		if err := rows.Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, nil
}

func (s *Store) AcceptReviewerTask(ctx context.Context, taskID string) (ReviewerTask, error) {
	var t ReviewerTask
	err := s.pool.QueryRow(ctx,
		`UPDATE reviewer_tasks SET status = 'accepted', accepted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND status = 'sent'
		 RETURNING id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		   sent_at, accepted_at, completed_at, expired_at, created_at, updated_at`, taskID).
		Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewerTask{}, ErrNotFound
	}
	return t, err
}

// CompleteReviewerTask marks a task completed and adds earnings to the reviewer.
func (s *Store) CompleteReviewerTask(ctx context.Context, taskID string) (ReviewerTask, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReviewerTask{}, err
	}
	defer tx.Rollback(ctx)

	var t ReviewerTask
	err = tx.QueryRow(ctx,
		`UPDATE reviewer_tasks SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND status IN ('sent','accepted')
		 RETURNING id, hitl_queue_id, reviewer_id, task_type, status, reward_amount,
		   sent_at, accepted_at, completed_at, expired_at, created_at, updated_at`, taskID).
		Scan(&t.ID, &t.HitlQueueID, &t.ReviewerID, &t.TaskType, &t.Status, &t.RewardAmount,
			&t.SentAt, &t.AcceptedAt, &t.CompletedAt, &t.ExpiredAt, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReviewerTask{}, ErrNotFound
	}
	if err != nil {
		return ReviewerTask{}, err
	}

	_, err = tx.Exec(ctx,
		`UPDATE reviewers SET total_earned = total_earned + $1, pending_payout = pending_payout + $1, updated_at = NOW()
		 WHERE id = $2`, t.RewardAmount, t.ReviewerID)
	if err != nil {
		return ReviewerTask{}, err
	}

	return t, tx.Commit(ctx)
}
