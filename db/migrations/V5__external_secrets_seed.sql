-- External Secrets Operator — seed initial secrets
-- These will be synced to Kubernetes Secrets by External Secrets Operator
-- WARNING: values here are placeholders; replace with real values in production

INSERT INTO external_secrets (name, value) VALUES
    ('fitpulse/production/app-secrets/JWT_PRIVATE_KEY_PEM',
     'REPLACE_WITH_REAL_JWT_PRIVATE_KEY_PEM'),
    ('fitpulse/production/app-secrets/JWT_PUBLIC_KEY_PEM',
     'REPLACE_WITH_REAL_JWT_PUBLIC_KEY_PEM'),
    ('fitpulse/production/app-secrets/RABBITMQ_URL',
     'REPLACE_WITH_REAL_RABBITMQ_URL'),
    ('fitpulse/production/app-secrets/VALKEY_PASSWORD',
     'REPLACE_WITH_REAL_VALKEY_PASSWORD'),
    ('fitpulse/production/app-secrets/POSTGRES_PASSWORD',
     'REPLACE_WITH_REAL_POSTGRES_PASSWORD'),
    ('fitpulse/production/app-secrets/GOOGLE_CLIENT_ID',
     'REPLACE_WITH_REAL_GOOGLE_CLIENT_ID'),
    ('fitpulse/production/app-secrets/GOOGLE_CLIENT_SECRET',
     'REPLACE_WITH_REAL_GOOGLE_CLIENT_SECRET'),
    ('fitpulse/production/app-secrets/SMTP_PASSWORD',
     'REPLACE_WITH_REAL_SMTP_PASSWORD'),
    ('fitpulse/production/app-secrets/TOTP_ENCRYPTION_KEY',
     'REPLACE_WITH_REAL_TOTP_ENCRYPTION_KEY')
ON CONFLICT (name) DO NOTHING;
