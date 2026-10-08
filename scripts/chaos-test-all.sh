#!/usr/bin/env bash
set -euo pipefail

# Master Chaos Test Script
# Runs all chaos tests and generates a report
#
# Usage: ./chaos-test-all.sh [namespace]
#   namespace - Kubernetes namespace (default: fitness-platform-production)

NAMESPACE="${1:-fitness-platform-production}"
REPORT_DIR="${REPORT_DIR:-./chaos-test-reports}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${REPORT_DIR}/chaos-test-report-${TIMESTAMP}.md"

mkdir -p "$REPORT_DIR"

echo "=== FitPulse Chaos Test Suite ==="
echo "Namespace: $NAMESPACE"
echo "Report: $REPORT_FILE"
echo ""

# Initialize report
cat >"$REPORT_FILE" <<EOF
# Chaos Test Report

**Date:** $(date -u +%Y-%m-%dT%H:%M:%SZ)  
**Namespace:** $NAMESPACE  
**Tester:** automated  

## Summary

| Test | Status | Recovery Time |
|------|--------|---------------|
| PostgreSQL | ⏳ Running | - |
| Valkey | ⏳ Running | - |
| Vault | ⏳ Running | - |

## Detailed Results

EOF

# Run PostgreSQL chaos test
echo "Running PostgreSQL chaos test..."
if bash "$(dirname "$0")/chaos-test-postgres.sh" "$NAMESPACE"; then
	PG_STATUS="✅ PASS"
	PG_TIME=$(bash "$(dirname "$0")/chaos-test-postgres.sh" "$NAMESPACE" 2>&1 | grep "Recovery time:" | awk '{print $3}' || echo "N/A")
else
	PG_STATUS="❌ FAIL"
	PG_TIME="N/A"
fi
echo "PostgreSQL: $PG_STATUS (${PG_TIME})"

# Run Valkey chaos test
echo "Running Valkey chaos test..."
if bash "$(dirname "$0")/chaos-test-valkey.sh" "$NAMESPACE"; then
	VALKEY_STATUS="✅ PASS"
	VALKEY_TIME=$(bash "$(dirname "$0")/chaos-test-valkey.sh" "$NAMESPACE" 2>&1 | grep "Recovery time:" | awk '{print $3}' || echo "N/A")
else
	VALKEY_STATUS="❌ FAIL"
	VALKEY_TIME="N/A"
fi
echo "Valkey: $VALKEY_STATUS (${VALKEY_TIME})"

# Run Vault chaos test
echo "Running Vault chaos test..."
if bash "$(dirname "$0")/chaos-test-vault.sh" "$NAMESPACE"; then
	VAULT_STATUS="✅ PASS"
	VAULT_TIME=$(bash "$(dirname "$0")/chaos-test-vault.sh" "$NAMESPACE" 2>&1 | grep "Recovery time:" | awk '{print $3}' || echo "N/A")
else
	VAULT_STATUS="❌ FAIL"
	VAULT_TIME="N/A"
fi
echo "Vault: $VAULT_STATUS (${VAULT_TIME})"

# Update report
cat >>"$REPORT_FILE" <<EOF

### PostgreSQL Failure

**Status:** $PG_STATUS  
**Recovery Time:** ${PG_TIME}s  

**Details:**
- Simulated deletion of PostgreSQL primary pod
- Verified pod recreation
- Verified service connectivity
- Verified data integrity

### Valkey Failure

**Status:** $VALKEY_STATUS  
**Recovery Time:** ${VALKEY_TIME}s  

**Details:**
- Simulated deletion of Valkey master pod
- Verified pod recreation
- Verified service connectivity
- Verified data persistence

### Vault Failure

**Status:** $VAULT_STATUS  
**Recovery Time:** ${VAULT_TIME}s  

**Details:**
- Simulated deletion of Vault pod
- Verified pod recreation
- Verified Vault accessibility

## Conclusion

EOF

# Overall result
if [[ "$PG_STATUS" == "✅ PASS" && "$VALKEY_STATUS" == "✅ PASS" && "$VAULT_STATUS" == "✅ PASS" ]]; then
	echo "✅ ALL TESTS PASSED" >>"$REPORT_FILE"
	echo ""
	echo "=== All chaos tests passed ==="
	exit 0
else
	echo "❌ SOME TESTS FAILED" >>"$REPORT_FILE"
	echo ""
	echo "=== Some chaos tests failed ==="
	exit 1
fi
