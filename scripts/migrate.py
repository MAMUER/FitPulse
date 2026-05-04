#!/usr/bin/env python3
"""
Fitness Platform — Database Migration Script (cross-platform)

Usage:
    python scripts/migrate.py
    DB_HOST=10.0.0.1 python scripts/migrate.py
"""

import os
import subprocess
import shutil
import sys
from pathlib import Path

# === Configuration ===
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASSWORD = os.getenv("DB_PASSWORD", "")
if not DB_PASSWORD:
    print("WARNING: DB_PASSWORD not set. Using default 'postgres' (dev only).")
    DB_PASSWORD = "postgres"
DB_NAME = os.getenv("DB_NAME", "fitness")

MIGRATIONS_DIR = Path("db/migrations")
DOCKER_COMPOSE_FILE = Path("deployments/docker-compose.yml")

GREEN = "\033[92m"
YELLOW = "\033[93m"
RED = "\033[91m"
CYAN = "\033[96m"
RESET = "\033[0m"
BOLD = "\033[1m"


def banner(text):
    print(f"\n{CYAN}{BOLD}{'=' * 48}{RESET}")
    print(f"{CYAN}{BOLD}   {text}{RESET}")
    print(f"{CYAN}{BOLD}{'=' * 48}{RESET}\n")


def info(text):
    print(f"{YELLOW}{text}{RESET}")


def success(text):
    print(f"{GREEN}  ✓ {text}{RESET}")


def error(text):
    print(f"{RED}  ✗ {text}{RESET}")


def run_psql(args, check=True):
    """Run psql command."""
    env = os.environ.copy()
    env["PGPASSWORD"] = DB_PASSWORD
    cmd = [
        "psql",
        "-h", DB_HOST,
        "-p", str(DB_PORT),
        "-U", DB_USER,
        "-d", DB_NAME,
        *args,
    ]
    result = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if check and result.returncode != 0:
        raise RuntimeError(f"psql failed: {result.stderr}")
    return result


def docker_compose_exec():
    """Run migration via Docker Compose exec."""
    cmd = [
        "docker", "compose",
        "-f", str(DOCKER_COMPOSE_FILE),
        "exec", "-T", "postgres",
        "psql", "-U", DB_USER, "-d", DB_NAME,
        "-f", "/docker-entrypoint-initdb.d/init.sql",
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result


def get_table_count():
    """Get number of tables in public schema."""
    result = run_psql([
        "-t", "-c",
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';"
    ])
    return result.stdout.strip()


def main():
    banner("DATABASE MIGRATIONS")

    info("Configuration:")
    print(f"  Host:     {DB_HOST}")
    print(f"  Port:     {DB_PORT}")
    print(f"  Database: {DB_NAME}")
    print(f"  User:     {DB_USER}")
    print()

    # Check if migrations directory exists
    if not MIGRATIONS_DIR.exists():
        error(f"Migrations directory not found: {MIGRATIONS_DIR}")
        sys.exit(1)

    # Get migration files
    migration_files = sorted(MIGRATIONS_DIR.glob("V*.sql"))
    if not migration_files:
        error(f"No migration files found in {MIGRATIONS_DIR}")
        sys.exit(1)

    # Strategy 1: Try psql directly
    psql_path = shutil.which("psql")
    if psql_path:
        info("[1/3] psql found, running migrations...")
        try:
            for mig_file in migration_files:
                info(f"Running migration: {mig_file.name}")
                result = run_psql(["-f", str(mig_file)], check=False)
                if result.returncode != 0:
                    error(f"Migration {mig_file.name} failed:\n{result.stderr}")
                    sys.exit(1)
            success("All migrations completed successfully")
        except Exception as e:
            error(f"psql error: {e}")
            sys.exit(1)
    else:
        info("[1/3] psql not found, using Docker...")
        if not DOCKER_COMPOSE_FILE.exists():
            error(f"Docker Compose file not found: {DOCKER_COMPOSE_FILE}")
            sys.exit(1)

        # For Docker, combine migrations into a single file and run
        combined_sql = Path("scripts/init-db.sql")
        info("[2/3] Combining migrations for Docker...")
        with open(combined_sql, 'w') as outfile:
            for mig_file in migration_files:
                outfile.write(f"-- {mig_file.name}\n")
                with open(mig_file, 'r') as infile:
                    outfile.write(infile.read())
                    outfile.write("\n\n")

        info("[2/3] Running combined migrations via Docker Compose...")
        cmd = [
            "docker", "compose",
            "-f", str(DOCKER_COMPOSE_FILE),
            "exec", "-T", "postgres",
            "psql", "-U", DB_USER, "-d", DB_NAME,
            "-f", "/docker-entrypoint-initdb.d/init.sql",
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode == 0:
            success("Migrations completed via Docker")
        else:
            error(f"Migrations failed:\n{result.stderr}")
            sys.exit(1)

    # Verify
    print()
    info("[3/3] Verifying migrations...")
    try:
        tables = get_table_count()
        success(f"Tables created: {tables}")
    except FileNotFoundError:
        info("Verification skipped (psql not installed)")
    except Exception as e:
        error(f"Verification failed: {e}")

    banner("MIGRATIONS COMPLETED!")


if __name__ == "__main__":
    main()
