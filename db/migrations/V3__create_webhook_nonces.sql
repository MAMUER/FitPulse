-- Webhook nonces table for replay-attack protection
CREATE TABLE IF NOT EXISTS webhook_nonces (
    user_id VARCHAR(255) NOT NULL,
    nonce VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, nonce)
);

-- Auto-purge old nonces after 24 hours
CREATE OR REPLACE FUNCTION purge_old_webhook_nonces()
RETURNS void AS $$
BEGIN
    DELETE FROM webhook_nonces
    WHERE created_at < NOW() - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;
