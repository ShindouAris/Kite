CREATE TABLE IF NOT EXISTS payment_sessions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    payment_id TEXT NOT NULL UNIQUE,
    app_id TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL,
    amount INTEGER NOT NULL,
    qr_image_url TEXT NOT NULL,
    qr_content TEXT NOT NULL,
    status TEXT NOT NULL,
    provider_transaction_id TEXT,
    raw_webhook_payload TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    paid_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS payment_sessions_app_id ON payment_sessions (app_id);
CREATE INDEX IF NOT EXISTS payment_sessions_status ON payment_sessions (status);
CREATE UNIQUE INDEX IF NOT EXISTS payment_sessions_provider_txn_unique
    ON payment_sessions (provider, provider_transaction_id)
    WHERE provider_transaction_id IS NOT NULL;