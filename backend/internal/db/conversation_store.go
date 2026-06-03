package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListConversations(ctx context.Context, tenantID string) ([]Conversation, error) {
	query := `SELECT id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
	           channel, COALESCE(line_user_id,''), status, created_at, updated_at
	           FROM conversations WHERE 1=1`
	args := []any{}
	if tenantID != "" {
		query += ` AND tenant_id = $1`
		args = append(args, tenantID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.TenantID, &c.BranchID, &c.UserID,
			&c.Channel, &c.LineUserID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

func (s *Store) GetConversation(ctx context.Context, id string) (Conversation, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
		  channel, COALESCE(line_user_id,''), status, created_at, updated_at
		  FROM conversations WHERE id = $1`, id).
		Scan(&c.ID, &c.TenantID, &c.BranchID, &c.UserID,
			&c.Channel, &c.LineUserID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateConversation(ctx context.Context, input Conversation) (Conversation, error) {
	if input.TenantID == "" {
		return Conversation{}, ErrInvalidInput
	}
	if input.Channel == "" {
		input.Channel = "liff"
	}
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`INSERT INTO conversations (tenant_id, branch_id, user_id, channel, line_user_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, COALESCE(branch_id::text,''), COALESCE(user_id::text,''),
		   channel, COALESCE(line_user_id,''), status, created_at, updated_at`,
		input.TenantID, nullIfEmpty(input.BranchID), nullIfEmpty(input.UserID),
		input.Channel, nullIfEmpty(input.LineUserID)).
		Scan(&c.ID, &c.TenantID, &c.BranchID, &c.UserID,
			&c.Channel, &c.LineUserID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Store) ListMessages(ctx context.Context, conversationID string) ([]Message, error) {
	if _, err := s.GetConversation(ctx, conversationID); err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, conversation_id, tenant_id, sender_type, COALESCE(sender_id::text,''),
		  message_type, content, metadata, created_at, updated_at
		  FROM messages WHERE conversation_id = $1 ORDER BY created_at`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.TenantID, &m.SenderType, &m.SenderID,
			&m.MessageType, &m.Content, &m.Metadata, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, nil
}

func (s *Store) CreateMessage(ctx context.Context, input Message) (Message, error) {
	if input.ConversationID == "" || input.Content == "" {
		return Message{}, ErrInvalidInput
	}
	conv, err := s.GetConversation(ctx, input.ConversationID)
	if err != nil {
		return Message{}, ErrNotFound
	}
	if input.MessageType == "" {
		input.MessageType = "text"
	}
	if input.SenderType == "" {
		input.SenderType = "admin"
	}
	var m Message
	err = s.pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, tenant_id, sender_type, sender_id, message_type, content, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, conversation_id, tenant_id, sender_type, COALESCE(sender_id::text,''),
		   message_type, content, metadata, created_at, updated_at`,
		input.ConversationID, conv.TenantID, input.SenderType, nullIfEmpty(input.SenderID),
		input.MessageType, input.Content, input.Metadata).
		Scan(&m.ID, &m.ConversationID, &m.TenantID, &m.SenderType, &m.SenderID,
			&m.MessageType, &m.Content, &m.Metadata, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
