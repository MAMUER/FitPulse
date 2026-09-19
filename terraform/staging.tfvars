# Staging variables
# Sensitive values are injected via environment variables in CI/CD
# or via -var flags in local development

vps_ip        = "YOUR_STAGING_VPS_IP_HERE"
domain        = "staging.fittpulse.ru"
storage_class = "local-path"

# Sensitive variables (do NOT commit real values here):
# - minio_root_user        -> TF_VAR_minio_root_user
# - minio_root_password    -> TF_VAR_minio_root_password
# - external_secrets_db_password -> TF_VAR_external_secrets_db_password
