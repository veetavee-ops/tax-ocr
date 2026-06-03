CREATE TABLE IF NOT EXISTS user_branches (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    branch_id  UUID        NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, branch_id)
);

CREATE INDEX idx_user_branches_user_id   ON user_branches (user_id);
CREATE INDEX idx_user_branches_branch_id ON user_branches (branch_id);
