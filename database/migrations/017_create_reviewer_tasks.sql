CREATE TABLE IF NOT EXISTS reviewer_tasks (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    hitl_queue_id UUID           NOT NULL REFERENCES hitl_queue (id) ON DELETE CASCADE,
    reviewer_id   UUID           NOT NULL REFERENCES reviewers (id),
    task_type     VARCHAR(30)    NOT NULL CHECK (task_type IN ('text_verification', 'classification_verification')),
    status        VARCHAR(20)    NOT NULL DEFAULT 'sent' CHECK (status IN ('sent', 'accepted', 'completed', 'expired')),
    reward_amount NUMERIC(10, 2) NOT NULL DEFAULT 0,
    sent_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    accepted_at   TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    expired_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reviewer_tasks_hitl_queue_id ON reviewer_tasks (hitl_queue_id);
CREATE INDEX idx_reviewer_tasks_reviewer_id   ON reviewer_tasks (reviewer_id);
CREATE INDEX idx_reviewer_tasks_status        ON reviewer_tasks (status);
CREATE INDEX idx_reviewer_tasks_task_type     ON reviewer_tasks (task_type);
