package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeProcessInvoice = "invoice:process"

type ProcessInvoicePayload struct {
	InvoiceID        string `json:"invoice_id"`
	DocumentImportID string `json:"document_import_id"`
	TenantID         string `json:"tenant_id"`
	BranchID         string `json:"branch_id"`
	FilePath         string `json:"file_path"`
	ContentType      string `json:"content_type"`
}

type Client struct {
	asynq *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{
		asynq: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (c *Client) Close() error {
	return c.asynq.Close()
}

func (c *Client) EnqueueProcessInvoice(ctx context.Context, payload ProcessInvoicePayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeProcessInvoice, data)
	_, err = c.asynq.EnqueueContext(ctx, task)
	return err
}
