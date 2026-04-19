#!/usr/bin/env python3
"""
Seed Admin User Script — FitPulse
==================================
Creates the initial admin user (email_confirmed = TRUE) and invite codes.
Reads configuration from .env file (same as the rest of the project).

Usage:
    python scripts/seed-admin.py
    python scripts/seed-admin.py --email admin@example.com --password MyPass123!
"""

import argparse
import os
import subprocess
import sys
from pathlib import Path

# No hardcoded passwords — all secrets come from .env or CLI args.
# The .env file MUST be in .gitignore (it is).


def load_env_file(env_path: Path) -> dict[str, str]:
    """Load .env file into dict (simple key=value parser)."""
    env_vars = {}
    if not env_path.exists():
        print(f"[WARN] .env file not found at {env_path}")
        return env_vars

    with open(env_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            env_vars[key] = value
    return env_vars


def generate_bcrypt_hash(password: str) -> str:
    """Generate bcrypt hash using Go (reuses project's Go toolchain)."""
    import tempfile

    go_code = """\
package main

import (
    "fmt"
    "os"

    "golang.org/x/crypto/bcrypt"
)

func main() {
    h, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Print(string(h))
}
"""
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".go", delete=False
    ) as f:
        f.write(go_code)
        go_file = f.name

    try:
        result = subprocess.run(
            ["go", "run", go_file, password],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            print(f"[ERROR] Failed to generate bcrypt hash: {result.stderr.strip()}")
            return ""
        return result.stdout
    finally:
        os.unlink(go_file)


def find_project_root() -> Path:
    """Find project root (where .env and scripts/ are)."""
    script_dir = Path(__file__).resolve().parent
    return script_dir.parent


def main():
    project_root = find_project_root()
    env_path = project_root / ".env"
    compose_file = project_root / "deployments" / "docker-compose.yml"

    # Load .env
    env_vars = load_env_file(env_path)

    # Environment variables take precedence over .env file
    for key, value in os.environ.items():
        if key.startswith("SEED_") or key in (
            "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"
        ):
            env_vars[key] = value

    # Configuration from .env (with fallback defaults — no secrets here)
    db_user = env_vars.get("DB_USER", "fitness_admin")
    db_name = env_vars.get("DB_NAME", "fitness")
    admin_email = env_vars.get("SEED_ADMIN_EMAIL", "admin@fitpulse.local")
    admin_password = env_vars.get("SEED_ADMIN_PASSWORD", "")
    admin_name = "System Administrator"

    # Parse CLI arguments (override .env / environment)
    parser = argparse.ArgumentParser(
        description="Create initial admin user and invite codes"
    )
    parser.add_argument("--email", help="Admin email (overrides .env)")
    parser.add_argument("--password", help="Admin password (overrides .env)")
    args = parser.parse_args()

    if args.email:
        admin_email = args.email
    if args.password:
        admin_password = args.password

    # Security: require password — never use a hardcoded default
    if not admin_password:
        print("[ERROR] Admin password is not set.")
        print("  Set SEED_ADMIN_PASSWORD in .env or pass --password PASSWORD")
        sys.exit(1)

    print()
    print("=" * 40)
    print("  FITPULSE -- ADMIN SEED SCRIPT")
    print("=" * 40)
    print()
    print(f"  Email:    {admin_email}")
    print(f"  DB:       {db_name}")
    print()

    # Check Docker is running
    try:
        subprocess.run(["docker", "ps"], check=True, capture_output=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("[ERROR] Docker is not running or not installed!")
        sys.exit(1)

    # Check PostgreSQL container exists
    try:
        result = subprocess.run(
            ["docker", "compose", "-f", str(compose_file), "--env-file", str(env_path), "ps", "-q", "postgres"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        postgres_cid = result.stdout.strip()
        if not postgres_cid:
            print("[ERROR] PostgreSQL container not found.")
            print("  Run: scripts/run-local.bat  (Windows)")
            print("  Run: ./scripts/run-local.sh (Linux/macOS)")
            sys.exit(1)
    except (subprocess.CalledProcessError, FileNotFoundError) as e:
        print(f"[ERROR] Docker compose failed: {e}")
        sys.exit(1)

    # Generate bcrypt hash via Go (avoids Python bcrypt dependency)
    print("[1/3] Generating bcrypt hash...")
    bcrypt_hash = generate_bcrypt_hash(admin_password)
    if not bcrypt_hash:
        sys.exit(1)
    print(f"     Hash generated: {bcrypt_hash[:20]}...")

    # Build SQL — write to stdin of psql to avoid shell-escaping issues with $
    print("[2/3] Creating admin user in database...")

    escaped_email = admin_email.replace("'", "''")
    escaped_name = admin_name.replace("'", "''")
    # Note: $ in bcrypt hash is passed via stdin, no escaping needed

    sql = f"""\
BEGIN;

-- Create admin user if not exists
INSERT INTO users (id, email, password_hash, full_name, role, email_confirmed, created_at, updated_at)
SELECT gen_random_uuid(), '{escaped_email}', %HASH%, '{escaped_name}', 'admin', TRUE, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = '{escaped_email}');

-- Create user profile if not exists
INSERT INTO user_profiles (user_id)
SELECT id FROM users WHERE email = '{escaped_email}'
AND NOT EXISTS (
    SELECT 1 FROM user_profiles
    WHERE user_id = (SELECT id FROM users WHERE email = '{escaped_email}')
);

-- Create bootstrap invite codes if not exist
INSERT INTO invite_codes (code, role, specialty, max_uses, is_active, created_at)
SELECT 'ADMIN-BOOTSTRAP-' || substr(md5(random()::text), 1, 8), 'admin', NULL, 10, TRUE, NOW()
WHERE NOT EXISTS (SELECT 1 FROM invite_codes WHERE code LIKE 'ADMIN-BOOTSTRAP-%');

-- Show created user
SELECT email, full_name, role, email_confirmed, created_at
FROM users WHERE email = '{escaped_email}';

-- Show invite codes
SELECT code, role, max_uses, is_active
FROM invite_codes WHERE code LIKE '%BOOTSTRAP%'
ORDER BY role;

COMMIT;
"""
    # Replace %HASH% placeholder with actual bcrypt hash (passed safely via psql variable)
    sql = sql.replace("%HASH%", f"'{bcrypt_hash}'")

    # Execute SQL via docker exec (stdin avoids shell escaping)
    try:
        result = subprocess.run(
            [
                "docker", "compose", "-f", str(compose_file), "--env-file", str(env_path),
                "exec", "-T", "postgres",
                "psql", "-U", db_user, "-d", db_name,
            ],
            input=sql,
            capture_output=True,
            text=True,
            timeout=30,
        )

        if result.stdout.strip():
            print(result.stdout)

        if result.returncode != 0:
            print(f"[ERROR] Seed script failed:\n{result.stderr}")
            sys.exit(1)

    except subprocess.TimeoutExpired:
        print("[ERROR] Database operation timed out")
        sys.exit(1)
    except Exception as e:
        print(f"[ERROR] Failed to execute SQL: {e}")
        sys.exit(1)

    print()
    print("=" * 40)
    print("  ADMIN USER READY")
    print("=" * 40)
    print()
    print(f"  Email:    {admin_email}")
    print("  Password: [set via .env or --password]")  # Never print the actual password
    print()
    print("  Login via browser:")
    print("    https://localhost:8443")
    print()
    print("  NOTE: Change the default password after first login!")
    print("=" * 40)


if __name__ == "__main__":
    main()
