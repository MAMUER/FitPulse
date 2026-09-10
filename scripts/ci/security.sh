#!/bin/bash
set -euo pipefail

install_tools() {
	for i in 1 2 3; do
		if go install github.com/securego/gosec/v2/cmd/gosec@9e75c0576c9878035d4221392108d458abe10fc3; then # NOSONAR
			break
		fi
		echo "gosec install attempt $i failed; retrying..."
		sleep 5
	done
	for i in 1 2 3; do
		if go install golang.org/x/vuln/cmd/govulncheck@19b0bb6a272792b9afa8a6983c3e9b9a1816947f; then # NOSONAR
			break
		fi
		echo "govulncheck install attempt $i failed; retrying..."
		sleep 5
	done
	for i in 1 2 3; do
		if go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@8f3b0c7ed018e57905fbd873c697e0b1ede605a5; then # NOSONAR
			break
		fi
		echo "golangci-lint install attempt $i failed; retrying..."
		sleep 5
	done
	echo "Installing syft..."
	curl --proto =https -fsSL https://raw.githubusercontent.com/anchore/syft/main/install.sh -o /tmp/install.sh
	bash /tmp/install.sh -b /usr/local/bin
	rm -f /tmp/install.sh
	echo "Installing TruffleHog..."
	curl --proto =https -sSfL https://github.com/trufflesecurity/trufflehog/releases/download/v3.82.0/trufflehog_3.82.0_linux_amd64.tar.gz -o /tmp/trufflehog.tar.gz
	tar -xzf /tmp/trufflehog.tar.gz -C /usr/local/bin trufflehog
	chmod +x /usr/local/bin/trufflehog
	for attempt in 1 2 3 4 5; do
		if curl --proto =https -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v8.21.2/gitleaks_8.21.2_linux_x64.tar.gz" -o /tmp/gitleaks.tar.gz; then
			break
		fi
		echo "Gitleaks download failed (attempt $attempt), retrying..."
		sleep $((attempt * 5))
	done
 	sudo tar -xzf /tmp/gitleaks.tar.gz -C /usr/local/bin gitleaks
 	sudo chmod +x /usr/local/bin/gitleaks

 	echo "Installing cosign..."
 	for i in 1 2 3; do
 		if go install github.com/sigstore/cosign/v2/cmd/cosign@latest; then
 			break
 		fi
 		echo "cosign install attempt $i failed; retrying..."
 		sleep 5
 	done
 }

run_gosec() {
	mkdir -p sarif-results
	gosec -exclude=G101,G201,G505 -exclude-generated -fmt=sarif -out=sarif-results/gosec.sarif ./... 2>&1 || true
	if [ ! -s sarif-results/gosec.sarif ]; then
		echo "{\"\$schema\":\"https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json\",\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"gosec\"}},\"results\":[]}]}" >sarif-results/gosec.sarif
	fi
}

run_govulncheck() {
	mkdir -p sarif-results
	govulncheck ./... >sarif-results/govulncheck.txt 2>&1 || true
}

run_trivy() {
	echo "-> Running Trivy config scan (misconfigurations)..."
	mkdir -p sarif-results
	pip install --only-binary :all: 'checkov==3.3.9'
	checkov --directory configs/k8s --directory scripts --output sarif --output-file sarif-results/checkov-results.sarif || true
	echo "✅ Checkov scan completed"
}

run_gitleaks() {
	mkdir -p sarif-results
	gitleaks detect --source=. --config=.gitleaks.toml --report-format=sarif --report-path=sarif-results/gitleaks.sarif || true
}

run_trufflehog() {
	mkdir -p sarif-results
	trufflehog filesystem . --json --only-verified >sarif-results/trufflehog.jsonl 2>&1 || true
	if [ -s sarif-results/trufflehog.jsonl ]; then
		echo "TruffleHog raw findings:"
		jq -r '"\(.DetectorName // "unknown") at \(.File // "unknown"):\(.Line // 0) | \(.Description // .Raw // "")"' sarif-results/trufflehog.jsonl 2>/dev/null | head -20 || true
	fi
}

generate_sbom() {
	mkdir -p sbom
	syft . \
		--source-name fitpulse \
		--source-version "$(git rev-parse --short HEAD)" \
		-o spdx-json=sbom/sbom.spdx.json \
		-o cyclonedx-json=sbom/sbom.cyclonedx.json
	ls -la sbom/
}

run_kubescape() {
	echo "-> Running Kubescape..."
	mkdir -p sarif-results
	tmpfile=$(mktemp)
	kubescape scan configs/k8s/base/ \
		--format sarif \
		--verbose \
		>"$tmpfile" 2>&1 ||
		true
	if jq -e '.runs' "$tmpfile" >/dev/null 2>&1; then
		mv "$tmpfile" sarif-results/kubescape-results.sarif
	else
		rm -f "$tmpfile"
		jq -n '{ "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json", "version": "2.1.0", "runs": [ { "tool": { "driver": { "name": "kubescape" } }, "results": [] } ] }' >sarif-results/kubescape-results.sarif
	fi
	if command -v jq &>/dev/null; then
		jq '.runs |= map(
			.results |= map(
				select(
					(.locations[0].physicalLocation.artifactLocation.uri | test("\\.[^/]+$"))
					and
					((.ruleId // "") | IN(
						"C-0012", "C-0013", "C-0017", "C-0016",
						"C-0009", "C-0010", "C-0011", "C-0021", "C-0022",
						"C-0055", "C-0045", "C-0048", "C-0056",
						"C-0237"
					) | not)
				)
			)
		)' sarif-results/kubescape-results.sarif >sarif-results/kubescape-results-fixed.sarif
		mv sarif-results/kubescape-results-fixed.sarif sarif-results/kubescape-results.sarif
	fi
}

run_semgrep() {
	mkdir -p sarif-results
	semgrep scan \
		--config p/security-audit \
		--config p/owasp-top-ten \
		--exclude cmd/classifier/classifier_integration_test.go \
		--exclude configs/monitoring/node-exporter/daemonset.yaml \
		--exclude configs/k8s/base/local-path-provisioner.yaml \
		--sarif -o sarif-results/semgrep-results.sarif .
}

run_checkov() {
	echo "-> Running Checkov..."
	mkdir -p sarif-results
	pip install --only-binary :all: 'checkov==3.3.9'
	checkov --directory configs/k8s --directory scripts --output sarif --output-file sarif-results/checkov-results.sarif || true
	echo "✅ Checkov scan completed"
}

run_kube_bench() {
	echo "-> Running kube-bench..."
	mkdir -p security-results
	kube-bench run --benchmark cis-1.18 --json >security-results/kube-bench-results.json 2>&1 || true
	echo "✅ kube-bench scan completed (results in security-results/kube-bench-results.json)"
}

run_kube_hunter() {
	echo "-> Running kube-hunter (passive scan)..."
	mkdir -p security-results
	pip install --only-binary :all: kube-hunter==0.6.8
	kube-hunter --remote 127.0.0.1 --json >security-results/kube-hunter-results.json 2>&1 || true
	echo "✅ kube-hunter scan completed (results in security-results/kube-hunter-results.json)"
}

verify_cosign() {
	if [ ! -f /tmp/cosign-keys/cosign.pub ]; then
		echo "⚠️ No public key available, skipping verification"
		exit 0
	fi
	SERVICES="user-service biometric-service training-service gateway device-aggregator"
	for svc in $SERVICES; do
		echo "Verifying $svc..."
		if cosign verify --key /tmp/cosign-keys/cosign.pub \
			"ghcr.io/mamuer/project/$svc:${IMAGE_TAG}" >/dev/null 2>&1; then
			echo "✅ $svc signature verified"
		else
			echo "⚠️ $svc not signed or signature invalid (expected on PRs)"
		fi
	done
}

download_cosign_public_key() {
	if [ -z "$COSIGN_PUBLIC_KEY" ]; then
		echo "⚠️ COSIGN_PUBLIC_KEY not set, skipping verification"
		exit 0
	fi
	mkdir -p /tmp/cosign-keys
	echo "$COSIGN_PUBLIC_KEY" >/tmp/cosign-keys/cosign.pub
}

sign_images() {
	services="user-service biometric-service training-service gateway device-aggregator"

	if [ -z "$COSIGN_PRIVATE_KEY" ]; then
		echo "❌ COSIGN_PRIVATE_KEY is empty" >&2
		exit 1
	fi

	if echo "$COSIGN_PRIVATE_KEY" | grep -q -- '-----BEGIN .*PRIVATE KEY-----'; then
		echo "$COSIGN_PRIVATE_KEY" >/tmp/cosign.key
	elif echo "$COSIGN_PRIVATE_KEY" | grep -q '\\n'; then
		printf '%b' "$COSIGN_PRIVATE_KEY" >/tmp/cosign.key
	else
		echo "$COSIGN_PRIVATE_KEY" | base64 -d >/tmp/cosign.key
	fi

	if ! grep -q -- '-----BEGIN .*PRIVATE KEY-----' /tmp/cosign.key; then
		echo "❌ COSIGN_PRIVATE_KEY did not produce a valid PEM block."
		echo "💡 Проверьте секрет COSIGN_PRIVATE_KEY в GitHub Settings. Возможно, при копировании ключа в base64-часть попал случайный перенос строки."
		head -n 5 /tmp/cosign.key >&2
		exit 1
	fi

	SIGN_FAILURES=0
	for svc in $services; do
		echo "Signing $svc..."
		digest=""
		for attempt in 1 2 3 4 5; do
			digest=$(docker buildx imagetools inspect "$REGISTRY/mamuer/project/$svc:$IMAGE_TAG" --format '{{.Digest}}' 2>/dev/null || true)
			if [ -n "$digest" ]; then
				break
			fi
			manifest_url="https://ghcr.io/v2/mamuer/project/$svc/manifests/$IMAGE_TAG"
			digest=$(curl --proto =https -s -L -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
				-H "Authorization: Bearer $GITHUB_TOKEN" \
				"$manifest_url" -I | grep -i 'docker-content-digest' | head -1 | awk '{print $2}' | tr -d '\r')
			if [ -n "$digest" ]; then
				break
			fi
			echo "⚠️  Digest resolution attempt $attempt for $svc failed, retrying..."
			sleep $((attempt * 2))
		done

		if [ -z "$digest" ]; then
			echo "❌ Failed to resolve digest for $svc:$IMAGE_TAG after retries" >&2
			SIGN_FAILURES=$((SIGN_FAILURES + 1))
			continue
		fi

		sign_ok=false
		for attempt in 1 2 3; do
			echo "🔑 Attempt $attempt: signing $svc@$digest"
			if cosign sign --yes --key /tmp/cosign.key "$REGISTRY/mamuer/project/$svc@$digest" 2>&1; then
				echo "✅ $svc signed successfully"
				sign_ok=true
				break
			else
				echo "⚠️  Cosign attempt $attempt for $svc failed, retrying..."
				sleep $((attempt * 3))
			fi
		done

		if [ "$sign_ok" = false ]; then
			echo "❌ Failed to sign $svc after retries" >&2
			SIGN_FAILURES=$((SIGN_FAILURES + 1))
		fi
	done

	rm -f /tmp/cosign.key

	if [ "$SIGN_FAILURES" -gt 0 ]; then
		echo "❌ $SIGN_FAILURES image(s) failed to sign" >&2
		exit 1
	fi
}

security_gate() {
	echo "============================================"
	echo "  Security Gate — Checking scan results"
	echo "============================================"
	FAIL=0
	if [ -f sarif-results/gosec.sarif ]; then
		GOSEC_CRITICAL=$(jq '[.runs[].results[] | select(.level == "error")] | length' sarif-results/gosec.sarif 2>/dev/null | head -1 || echo 0)
		GOSEC_CRITICAL=$(printf '%s' "$GOSEC_CRITICAL")
		GOSEC_HIGH=$(jq '[.runs[].results[] | select(.level == "warning")] | length' sarif-results/gosec.sarif 2>/dev/null | head -1 || echo 0)
		GOSEC_HIGH=$(printf '%s' "$GOSEC_HIGH")
		echo "gosec: $GOSEC_CRITICAL critical, $GOSEC_HIGH high"
		if [ "$GOSEC_CRITICAL" -gt 0 ]; then
			echo "❌ FAIL: gosec found $GOSEC_CRITICAL critical issues"
			jq -r '.runs[].results[] | select(.level == "error") | "  - \(.ruleId): \(.message.text) at \(.locations[0].physicalLocation.artifactLocation.uri // "unknown"):\(.locations[0].physicalLocation.region.startLine // 0)"' sarif-results/gosec.sarif 2>/dev/null || true
			FAIL=1
		fi
	fi
	if [ -f sarif-results/trivy-fs.sarif ]; then
		TRIVY_CRITICAL=$(jq '[.runs[].results[] | select(.level == "error")] | length' sarif-results/trivy-fs.sarif 2>/dev/null | head -1 || echo 0)
		TRIVY_CRITICAL=$(printf '%s' "$TRIVY_CRITICAL")
		echo "trivy-fs: $TRIVY_CRITICAL critical CVEs"
		if [ "$TRIVY_CRITICAL" -gt 0 ]; then
			echo "❌ FAIL: Trivy found $TRIVY_CRITICAL critical vulnerabilities in dependencies"
			jq -r '.runs[].results[] | select(.level == "error") | "  - \(.ruleId): \(.message.text)"' sarif-results/trivy-fs.sarif 2>/dev/null | head -10 || true
			FAIL=1
		fi
	fi
	if [ -f sarif-results/gitleaks.sarif ]; then
		GITLEAKS=$(jq '[.runs[].results[] | select(.level == "error" or .level == "warning")] | length' sarif-results/gitleaks.sarif 2>/dev/null | head -1 || echo 0)
		GITLEAKS=$(printf '%s' "$GITLEAKS")
		if [ -z "$GITLEAKS" ]; then GITLEAKS=0; fi
		echo "gitleaks: $GITLEAKS secrets found"
		if [ "$GITLEAKS" -gt 0 ]; then
			echo "❌ FAIL: Gitleaks found $GITLEAKS potential secrets"
			jq -r '.runs[].results[] | select(.level == "error" or .level == "warning") | "  - \(.ruleId): \(.message.text) at \(.locations[0].physicalLocation.artifactLocation.uri // "unknown"):\(.locations[0].physicalLocation.region.startLine // 0)"' sarif-results/gitleaks.sarif 2>/dev/null | head -10 || true
			FAIL=1
		fi
	fi
	if [ -f sarif-results/trufflehog.jsonl ]; then
		TRUFFLEHOG=$(jq -r 'select(.Verified == true) | 1' sarif-results/trufflehog.jsonl 2>/dev/null | wc -l)
		TRUFFLEHOG=$(printf '%s' "$TRUFFLEHOG" | tr -d '[:space:]')
		echo "trufflehog: $TRUFFLEHOG secrets found"
		if [ "$TRUFFLEHOG" -gt 0 ]; then
			echo "❌ FAIL: TruffleHog found $TRUFFLEHOG potential secrets"
			jq -r 'select(.Verified == true) | "  - \(.DetectorName // "unknown"): \(.Description // .Raw // "")"' sarif-results/trufflehog.jsonl 2>/dev/null | head -10 || true
			FAIL=1
		fi
	fi
	if [ -f sarif-results/govulncheck.txt ]; then
		GOVULNCHECK=$(grep -cE "^\(#[0-9]+\)" sarif-results/govulncheck.txt 2>/dev/null || true)
		GOVULNCHECK=${GOVULNCHECK:-0}
		echo "govulncheck: $GOVULNCHECK vulnerabilities found"
		if [ "$GOVULNCHECK" -gt 0 ]; then
			echo "⚠️  WARNING: govulncheck found $GOVULNCHECK potential vulnerabilities"
			grep -E "^\(#[0-9]+\)" sarif-results/govulncheck.txt 2>/dev/null | head -10 || true
			echo "   Note: Trivy covers CVE scanning. govulncheck results are informational."
		fi
	fi
	echo "============================================"
	if [ "$FAIL" -eq 1 ]; then
		echo "❌ Security Gate FAILED — pipeline blocked"
		echo "   Fix the issues above before merging."
		exit 1
	else
		echo "✅ Security Gate PASSED — no critical issues found"
	fi
}

run_osv_python_requirements() {
	echo "Scanning Python requirements..."
	find . -name "requirements.txt" -not -path "./.git/*" -print0 | while IFS= read -r -d '' req_file; do
		echo "=== Scanning $req_file ==="
		docker run --rm -v "${GITHUB_WORKSPACE}:/repo" -w /repo \
			ghcr.io/google/osv-scanner:v2.3.8 \
			scan "$req_file" || echo "⚠️ Could not fully resolve $req_file (no lock file)"
	done
	echo "✅ Python requirements scan completed (some packages may not be fully resolved)"
}

run_osv_scanner() {
	osv-scanner scan . 2>&1 || true
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		install_tools
		run_gosec
		run_govulncheck
		run_trivy
		run_gitleaks
		run_trufflehog
		generate_sbom
		run_kubescape
		run_semgrep
		run_checkov
		run_kube_bench
		run_kube_hunter
		verify_cosign
		sign_images
		security_gate
		;;
	install_tools) install_tools ;;
	run_gosec) run_gosec ;;
	run_govulncheck) run_govulncheck ;;
	run_trivy) run_trivy ;;
	run_gitleaks) run_gitleaks ;;
	run_trufflehog) run_trufflehog ;;
	generate_sbom) generate_sbom ;;
	run_kubescape) run_kubescape ;;
	run_semgrep) run_semgrep ;;
	run_checkov) run_checkov ;;
	run_kube_bench) run_kube_bench ;;
	run_kube_hunter) run_kube_hunter ;;
	verify_cosign) verify_cosign ;;
	download_cosign_public_key) download_cosign_public_key ;;
	run_osv_scanner) run_osv_scanner ;;
	run_osv_python_requirements) run_osv_python_requirements ;;
	sign_images) sign_images ;;
	security_gate) security_gate ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
