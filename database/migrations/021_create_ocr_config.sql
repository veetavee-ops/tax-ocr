CREATE TABLE IF NOT EXISTS ocr_config (
    id          UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    provider    VARCHAR(50)  NOT NULL,
    api_key     TEXT         NOT NULL DEFAULT '',
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    updated_by  UUID         REFERENCES users(id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT ocr_config_provider_unique UNIQUE (provider)
);

INSERT INTO ocr_config (provider, api_key, enabled)
VALUES ('openai', '', true), ('gcv', '', true)
ON CONFLICT (provider) DO NOTHING;
