#!/usr/bin/env pwsh
<#
.SYNOPSIS
Run Terraform plan/apply for FitPulse infrastructure.
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("init", "plan", "apply", "destroy")]
    [string]$Command = "plan",

    [Parameter(Mandatory=$false)]
    [string]$VarFile = "production.tfvars"
)

$ErrorActionPreference = "Stop"
$TerraformDir = Join-Path $PSScriptRoot "terraform"

Set-Location -LiteralPath $TerraformDir

switch ($Command) {
    "init" {
        terraform init
    }
    "plan" {
        terraform plan -var-file=$VarFile
    }
    "apply" {
        terraform apply -var-file=$VarFile -auto-approve
    }
    "destroy" {
        terraform destroy -var-file=$VarFile -auto-approve
    }
}
