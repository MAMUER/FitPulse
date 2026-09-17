#!/usr/bin/env pwsh
<#
.SYNOPSIS
Setup External Secrets Operator PostgreSQL backend on Windows/PowerShell.
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$PostgresHost = "localhost",
    
    [Parameter(Mandatory=$true)]
    [string]$PostgresUser = "postgres",
    
    [Parameter(Mandatory=$true)]
    [string]$ExternalSecretsPassword = "REPLACE_WITH_REAL_PASSWORD",
    
    [Parameter(Mandatory=$true)]
    [string]$Database = "fitness_platform"
)

$ErrorActionPreference = "Stop"

Write-Host "Creating external_secrets user and granting permissions..."

$sql = @"
CREATE USER external_secrets WITH PASSWORD '$ExternalSecretsPassword';
GRANT USAGE ON SCHEMA public TO external_secrets;
GRANT SELECT, INSERT, UPDATE, DELETE ON external_secrets TO external_secrets;
GRANT USAGE, SELECT ON SEQUENCE external_secrets_id_seq TO external_secrets;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON DATABASE $Database FROM PUBLIC;
"@

$sql | & psql -h $PostgresHost -U $PostgresUser -d $Database

Write-Host "External Secrets PostgreSQL backend setup complete."
