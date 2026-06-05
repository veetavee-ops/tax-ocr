package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListRewardConfigs(ctx context.Context) ([]RewardConfig, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, task_type, amount, currency, COALESCE(updated_by::text,''), created_at, updated_at
		 FROM reward_config ORDER BY task_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RewardConfig
	for rows.Next() {
		var rc RewardConfig
		if err := rows.Scan(&rc.ID, &rc.TaskType, &rc.Amount, &rc.Currency,
			&rc.UpdatedBy, &rc.CreatedAt, &rc.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, rc)
	}
	return items, nil
}

func (s *Store) GetRewardAmounts(ctx context.Context) (map[string]float64, error) {
	configs, err := s.ListRewardConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(configs))
	for _, c := range configs {
		out[c.TaskType] = c.Amount
	}
	return out, nil
}

func (s *Store) UpdateRewardConfig(ctx context.Context, id string, amount float64, updatedBy string) (RewardConfig, error) {
	if amount <= 0 {
		return RewardConfig{}, ErrInvalidInput
	}
	var rc RewardConfig
	err := s.pool.QueryRow(ctx,
		`UPDATE reward_config SET amount = $2, updated_by = $3, updated_at = NOW() WHERE id = $1
		 RETURNING id, task_type, amount, currency, COALESCE(updated_by::text,''), created_at, updated_at`,
		id, amount, nullIfEmpty(updatedBy)).
		Scan(&rc.ID, &rc.TaskType, &rc.Amount, &rc.Currency,
			&rc.UpdatedBy, &rc.CreatedAt, &rc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RewardConfig{}, ErrNotFound
	}
	return rc, err
}
