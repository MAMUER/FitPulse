#!/usr/bin/env bash
set -euo pipefail

DOMAIN="${CSP_DOMAIN:-https://fittpulse.duckdns.org}"
EXPECTED_HEADERS=(
	"Content-Security-Policy"
	"Cross-Origin-Opener-Policy"
	"Cross-Origin-Embedder-Policy"
	"X-Frame-Options"
	"X-Content-Type-Options"
	"X-XSS-Protection"
	"Referrer-Policy"
	"Permissions-Policy"
)

echo "============================================"
echo "  CSP Headers Check"
echo "  Domain: $DOMAIN"
echo "============================================"

HTTP_CODE=""
for attempt in {1..6}; do
	if CURL_OUTPUT=$(curl -sS -o /dev/null -w "HTTP %{http_code}" \
		--connect-timeout 5 --max-time 10 "$DOMAIN" 2>&1); then
		HTTP_CODE="${CURL_OUTPUT#HTTP }"
		if [[ "$HTTP_CODE" == "200" ]]; then
			break
		fi
		echo "Attempt $attempt/6: site returned $CURL_OUTPUT"
	else
		echo "Attempt $attempt/6: request failed: $CURL_OUTPUT"
	fi
	if [[ "$attempt" -lt 6 ]]; then
		sleep 10
	fi
done
if [[ "$HTTP_CODE" != "200" ]]; then
	echo "::error::Site did not return HTTP 200 after 6 attempts (last status: ${HTTP_CODE:-unavailable})"
	exit 1
fi
echo "✅ Site is accessible (HTTP 200)"

if ! HEADERS=$(curl -sS -I --connect-timeout 5 --max-time 15 "$DOMAIN"); then
	echo "::error::Could not retrieve response headers from $DOMAIN"
	exit 1
fi
MISSING=()

for header in "${EXPECTED_HEADERS[@]}"; do
	if ! echo "$HEADERS" | grep -qi "^$header:"; then
		MISSING+=("$header")
	fi
done

if [[ ${#MISSING[@]} -gt 0 ]]; then
	echo "❌ FAIL: Missing headers:"
	for h in "${MISSING[@]}"; do
		echo "  - $h"
	done
	exit 1
fi
echo "✅ All expected security headers present"

CSP_VALUE=$(echo "$HEADERS" | grep -i "^Content-Security-Policy:" | cut -d' ' -f2- | tr -d '\r')
echo ""
echo "Content-Security-Policy:"
echo "  $CSP_VALUE"
echo ""

REQUIRED_DIRECTIVES=(
	"default-src"
	"script-src"
	"style-src"
	"img-src"
	"connect-src"
	"frame-ancestors"
)

MISSING_DIRECTIVES=()
for directive in "${REQUIRED_DIRECTIVES[@]}"; do
	if ! echo "$CSP_VALUE" | grep -qi "$directive"; then
		MISSING_DIRECTIVES+=("$directive")
	fi
done

if [[ ${#MISSING_DIRECTIVES[@]} -gt 0 ]]; then
	echo "⚠️  WARNING: Missing CSP directives:"
	for d in "${MISSING_DIRECTIVES[@]}"; do
		echo "  - $d"
	done
else
	echo "✅ All required CSP directives present"
fi

if echo "$CSP_VALUE" | grep -qi "'unsafe-inline'"; then
	echo "⚠️  WARNING: CSP contains 'unsafe-inline'"
fi

if echo "$CSP_VALUE" | grep -qi "'unsafe-eval'"; then
	echo "⚠️  WARNING: CSP contains 'unsafe-eval'"
fi

COOP_VALUE=$(echo "$HEADERS" | grep -i "^Cross-Origin-Opener-Policy:" | cut -d' ' -f2- | tr -d '\r')
COEP_VALUE=$(echo "$HEADERS" | grep -i "^Cross-Origin-Embedder-Policy:" | cut -d' ' -f2- | tr -d '\r')

echo ""
echo "Cross-Origin-Opener-Policy: $COOP_VALUE"
echo "Cross-Origin-Embedder-Policy: $COEP_VALUE"

if [[ "$COOP_VALUE" = "same-origin" ]] && [[ "$COEP_VALUE" = "require-corp" ]]; then
	echo "✅ COOP/COEP configured for cross-origin isolation"
else
	echo "⚠️  WARNING: COOP/COEP not set to full isolation values"
fi

echo ""
echo "✅ CSP Headers Check PASSED"
echo "   Run https://csp-evaluator.withgoogle.com/ for detailed analysis"
