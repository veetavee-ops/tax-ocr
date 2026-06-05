package archive

import (
	"context"
	"fmt"
	"log"
	"time"

	"tax-ocr/backend/internal/db"
)

type Scheduler struct {
	store    *db.Store
	interval time.Duration
}

func NewScheduler(store *db.Store, interval time.Duration) *Scheduler {
	return &Scheduler{store: store, interval: interval}
}

// Run starts the archive loop in a background goroutine.
func (sc *Scheduler) Run(ctx context.Context) {
	go func() {
		sc.runOnce(ctx)
		ticker := time.NewTicker(sc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sc.runOnce(ctx)
			}
		}
	}()
}

func (sc *Scheduler) runOnce(ctx context.Context) {
	policies, err := sc.store.ListArchivePolicies(ctx, "")
	if err != nil {
		log.Printf("[archive] list policies: %v", err)
		return
	}
	if len(policies) == 0 {
		return
	}

	total := 0
	for _, p := range policies {
		n, err := sc.archiveTenant(ctx, p)
		if err != nil {
			log.Printf("[archive] tenant %s: %v", p.TenantID, err)
			continue
		}
		total += n
	}
	if total > 0 {
		log.Printf("[archive] archived %d invoices", total)
	}
}

func (sc *Scheduler) archiveTenant(ctx context.Context, policy db.ArchivePolicy) (int, error) {
	invoices, err := sc.store.FindInvoicesToArchive(ctx, policy.TenantID, policy.ActiveDays)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, inv := range invoices {
		archivePath := fmt.Sprintf("archive/%s/%d/%s", inv.TenantID, inv.CreatedAt.Year(), inv.FilePath)

		if _, err := sc.store.CreateArchiveLog(ctx, db.ArchiveLog{
			TenantID:    inv.TenantID,
			EntityType:  "invoice",
			EntityID:    inv.ID,
			ArchivePath: archivePath,
		}); err != nil {
			log.Printf("[archive] create log invoice %s: %v", inv.ID, err)
			continue
		}

		if err := sc.store.MarkInvoiceArchived(ctx, inv.ID); err != nil {
			log.Printf("[archive] mark archived invoice %s: %v", inv.ID, err)
			continue
		}
		count++
	}
	return count, nil
}
