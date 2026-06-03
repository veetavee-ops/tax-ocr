CREATE TABLE IF NOT EXISTS reviewer_payouts (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    reviewer_id    UUID           NOT NULL REFERENCES reviewers (id),
    amount         NUMERIC(12, 2) NOT NULL,
    method         VARCHAR(20)    NOT NULL CHECK (method IN ('promptpay', 'bank')),
    account_number VARCHAR(20)    NOT NULL,
    status         VARCHAR(20)    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid')),
    paid_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reviewer_payouts_reviewer_id ON reviewer_payouts (reviewer_id);
CREATE INDEX idx_reviewer_payouts_status      ON reviewer_payouts (status);
