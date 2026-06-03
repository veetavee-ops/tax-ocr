CREATE TABLE IF NOT EXISTS reviewers (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(255)   NOT NULL,
    line_user_id   VARCHAR(100)   NOT NULL UNIQUE,
    reviewer_type  VARCHAR(30)    NOT NULL CHECK (reviewer_type IN ('text_verifier', 'classification_verifier')),
    status         VARCHAR(20)    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    total_earned   NUMERIC(12, 2) NOT NULL DEFAULT 0,
    pending_payout NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reviewers_line_user_id  ON reviewers (line_user_id);
CREATE INDEX idx_reviewers_reviewer_type ON reviewers (reviewer_type);
CREATE INDEX idx_reviewers_status        ON reviewers (status);
