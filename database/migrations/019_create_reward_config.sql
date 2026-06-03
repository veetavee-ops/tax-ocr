CREATE TABLE IF NOT EXISTS reward_config (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type  VARCHAR(30)    NOT NULL UNIQUE CHECK (task_type IN ('text_verification', 'classification_verification')),
    amount     NUMERIC(10, 2) NOT NULL DEFAULT 0,
    currency   VARCHAR(3)     NOT NULL DEFAULT 'THB',
    updated_by UUID           REFERENCES users (id),
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- seed default reward config
INSERT INTO reward_config (task_type, amount, currency)
VALUES
    ('text_verification',          5.00, 'THB'),
    ('classification_verification', 3.00, 'THB')
ON CONFLICT (task_type) DO NOTHING;
