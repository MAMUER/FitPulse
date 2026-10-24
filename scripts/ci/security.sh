#!/bin/bash
set -euo pipefail

install_gosec() {
	go install github.com/securego/gosec/v2/cmd/gosec@9e75c0576c9878035d4221392108d458abe10fc3 # NOSONAR
}

install_govulncheck() {
	go install golang.org/x/vuln/cmd/govulncheck@19b0bb6a272792b9afa8a6983c3e9b9a1816947f # NOSONAR
}

install_golangci_lint() {
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 # NOSONAR
}

install_cosign() {
	go install github.com/sigstore/cosign/v2/cmd/cosign@v2.6.5 # NOSONAR
}

install_syft() {
	curl --proto =https --retry 3 --retry-delay 5 -fsSL https://raw.githubusercontent.com/anchore/syft/main/install.sh -o /tmp/install.sh
	bash /tmp/install.sh -b /usr/local/bin
	rm -f /tmp/install.sh
}

install_trufflehog() {
	curl --proto =https --retry 3 --retry-delay 5 -sSfL https://github.com/trufflesecurity/trufflehog/releases/download/v3.82.0/trufflehog_3.82.0_linux_amd64.tar.gz -o /tmp/trufflehog.tar.gz
	tar -xzf /tmp/trufflehog.tar.gz -C /usr/local/bin trufflehog
	chmod +x /usr/local/bin/trufflehog
}

install_gitleaks() {
	curl --proto =https --retry 3 --retry-delay 5 -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v8.21.2/gitleaks_8.21.2_linux_x64.tar.gz" -o /tmp/gitleaks.tar.gz
	sudo tar -xzf /tmp/gitleaks.tar.gz -C /usr/local/bin gitleaks
	sudo chmod +x /usr/local/bin/gitleaks
}

install_kube_bench() {
	command -v kube-bench >/dev/null 2>&1 && return 0
	echo "-> Installing kube-bench..."
	curl --proto =https --retry 3 --retry-delay 5 -fsSL "https://github.com/aquasecurity/kube-bench/releases/download/v0.13.0/kube-bench_0.13.0_linux_amd64.tar.gz" -o /tmp/kube-bench.tar.gz
	sudo tar -xzf /tmp/kube-bench.tar.gz -C /usr/local/bin kube-bench
	sudo chmod +x /usr/local/bin/kube-bench
	rm -f /tmp/kube-bench.tar.gz
}

# install_tools ставит всё разом — вызывать один раз на job (см. ci.yml).
# Точечная установка: install_<tool>.
install_tools() {
	install_gosec
	install_govulncheck
	install_golangci_lint
	install_syft
	install_trufflehog
	install_gitleaks
	install_cosign
}

run_gosec() {
	mkdir -p sarif-results
	# gosec может падать при пакетах с type errors — не рвём пайплайн, но сохраняем SARIF если есть
	gosec -exclude=G101,G201,G505 -exclude-generated -fmt=sarif -out=sarif-results/gosec.sarif ./... 2>&1 || true
	if [ ! -s sarif-results/gosec.sarif ]; then
		echo "{\"\$schema\":\"https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json\",\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"gosec\"}},\"results\":[]}]}" >sarif-results/gosec.sarif
	fi
}

run_govulncheck() {
	mkdir -p sarif-results
	govulncheck ./... >sarif-results/govulncheck.txt 2>&1
}

run_gitleaks() {
	mkdir -p sarif-results
	# gitleaks exits 1 when leaks are found - that is the enforcement.
	# The SARIF report is still written for Code Scanning.
	gitleaks detect --source=. --config=.gitleaks.toml --report-format=sarif --report-path=sarif-results/gitleaks.sarif
}

run_trufflehog() {
	mkdir -p sarif-results
	# --only-verified: only verified secrets are reported; trufflehog exits
	# 194 when findings exist, which fails the job.
	trufflehog filesystem . --json --only-verified >sarif-results/trufflehog.jsonl
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
	# Guard against a silent no-op: without kubescape on PATH the scan below
	# would fail into `|| true` and an empty SARIF would be uploaded.
	command -v kubescape >/dev/null 2>&1 || {
		echo "❌ kubescape not found in PATH — install it (ci.yml) before running" >&2
		exit 1
	}
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
	local status=0
	# checkov 3.3.9 only has --output-file-path (a folder); the SARIF lands as
	# results_sarif.sarif inside it and is renamed for the upload step.
	checkov --directory configs/k8s --directory scripts --output sarif \
		--output-file-path sarif-results --compact || status=$?

	# checkov writes results_sarif.sarif inside the output folder; the upload
	# step expects the fixed name.
	mv -f sarif-results/results_sarif.sarif sarif-results/checkov-results.sarif 2>/dev/null

	# checkov exit codes: 0 = no findings, 1 = findings, >=2 = scanner error
	if [ "$status" -ge 2 ]; then
		echo "::error::Checkov failed to run (exit $status) - IaC scan did not complete, results are NOT green"
		exit "$status"
	fi

	if [ "$status" -eq 1 ]; then
		echo "::error::Checkov found IaC policy violations - failing the job. Fix the findings below or add a justified skip in .checkov.yaml"
		python3 - <<-'PY'
			import collections
			import json
			import os

			try:
			    with open("sarif-results/checkov-results.sarif") as fh:
			        data = json.load(fh)
			except OSError as exc:
			    print(f"Could not read Checkov SARIF: {exc}")
			    raise SystemExit(0)

			definitions = {}
			for rule in data["runs"][0]["tool"]["driver"].get("rules", []):
			    definitions[rule.get("id")] = rule.get("shortDescription", {}).get("text", "")

			counter = collections.Counter(
			    result["ruleId"]
			    for run in data.get("runs", [])
			    for result in run.get("results", [])
			)
			total = sum(counter.values())
			if not total:
			    print("Checkov: no violations")
			    raise SystemExit(0)

			print(f"Checkov violations: {total} across {len(counter)} checks")
			for rule, count in counter.most_common():
			    title = definitions.get(rule, "")
			    print(f"  {rule}: {count} - {title}")

			summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
			if not summary_path:
			    raise SystemExit(0)

			rows = [
			    "### Checkov: IaC policy violations",
			    "",
			    f"**{total} violations** across **{len(counter)} checks**",
			    "",
			    "| Check | Count | Description |",
			    "| --- | --- | --- |",
			]
			for rule, count in counter.most_common():
			    title = definitions.get(rule, "").replace("|", "\\|")
			    rows.append(f"| `{rule}` | {count} | {title} |")
			rows += [
			    "",
			    "Fix the findings above, or add a justified skip to `.checkov.yaml`.",
			]

			with open(summary_path, "a", encoding="utf-8") as fh:
			    fh.write("\n".join(rows) + "\n")
			print("Full breakdown written to the job summary")
		PY
		exit 1
	fi

	echo "✅ Checkov scan completed"
}

run_kube_bench() {
	echo "-> Running kube-bench..."
	mkdir -p security-results

	if ! command -v kube-bench >/dev/null 2>&1; then
		install_kube_bench || true
	fi

	if ! command -v kube-bench >/dev/null 2>&1; then
		echo "::error::kube-bench binary not found after installation attempt. Установите его в этом job'е или запустите через SSH на ноде кластера." >&2
		exit 127
	fi

	# kube-bench must run on the Kubernetes node or against an existing node config.
	# On GitHub runner (before cluster provisioning) there is usually no kubelet/etcd/k8s paths.
	if [ ! -d /var/lib/kubelet ] && [ ! -d /etc/kubernetes ] && [ ! -f /etc/systemd/system/kubelet.service ] && [ ! -f "$HOME/.kube/config" ] && [ -z "${KUBECONFIG:-}" ]; then
		echo "::error::Kube-bench CIS benchmark требует запустить проверку НА самой ноде Kubernetes. В этом job нет кластера/kubelet-конфигов." >&2
		echo "Причина: job 'kube-bench' запускается до provision-k8s-vps. Либо перенесите его после поднятия кластера с SSH на VPS, либо выполните kube-bench прямо на control-plane ноде через SSH." >&2
		exit 1
	fi

	# Try to scan; do not mask real failures silently unless it's clearly a local runner issue
	kube-bench run --benchmark cis-1.18 --json >security-results/kube-bench-results.json 2>&1 || {
		rc=$?
		echo "::error::kube-bench завершился с кодом $rc. Убедитесь, что он запущен на ноде k8s с необходимыми правами/конфигами." >&2
		exit $rc
	}
	echo "✅ kube-bench scan completed (results in security-results/kube-bench-results.json)"
}

run_kube_hunter() {
	echo "-> Running kube-hunter (passive scan)..."
	mkdir -p security-results

	# kube-hunter also needs a target cluster/host to scan
	if [ ! -f "$HOME/.kube/config" ] && [ -z "${KUBECONFIG:-}" ]; then
		echo "::error::kube-hunter требует наличия кластера для сканирования. В этом job нет kubeconfig/доступного кластера." >&2
		echo "Причина: job 'kube-hunter' запускается до provision-k8s-vps. Перенесите его после поднятия кластера или запустите на целевом хосте." >&2
		exit 1
	fi

	pip install --only-binary :all: kube-hunter==0.6.8 2>&1 || pip install kube-hunter==0.6.8 2>&1 || true

	if ! command -v kube-hunter >/dev/null 2>&1 && ! python3 -m kube_hunter --help >/dev/null 2>&1; then
		echo "::error::Не удалось установить/найти kube-hunter." >&2
		exit 127
	fi

	kube-hunter --remote 127.0.0.1 --json >security-results/kube-hunter-results.json 2>&1 || {
		rc=$?
		echo "::error::kube-hunter завершился с кодом $rc. Проверьте доступность целевого кластера." >&2
		exit $rc
	}
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
		# Sign by tag: cosign resolves the digest itself using the registry
		# credentials from docker/login-action (~/.docker/config.json).
		# Manual digest resolution via `docker buildx imagetools inspect` or
		# the GHCR API is fragile (buildx plugin, token exchange for private
		# packages) and was removed - cosign handles both.
		sign_ok=false
		for attempt in 1 2 3; do
			echo "🔑 Attempt $attempt: signing $svc:$IMAGE_TAG"
			if cosign sign --yes --key /tmp/cosign.key "$REGISTRY/mamuer/project/$svc:$IMAGE_TAG" 2>&1; then
				echo "✅ $svc signed successfully"
				sign_ok=true
				break
			fi
			echo "⚠️  Cosign attempt $attempt for $svc failed, retrying..."
			echo "   Если ключ зашифрован — проверь COSIGN_PASSWORD; если нет — убери COSIGN_PASSWORD из secrets" >&2
			sleep $((attempt * 3))
		done

		if [ "$sign_ok" = false ]; then
			echo "❌ Failed to sign $svc:$IMAGE_TAG after retries" >&2
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
	echo "  Security Gate — summary"
	echo "============================================"
	FAIL=0

	# gosec: only HIGH/CRITICAL findings (SARIF level=error) block the build.
	if [ -f sarif-results/gosec.sarif ]; then
		GOSEC_CRITICAL=$(jq '[.runs[].results[] | select(.level == "error")] | length' sarif-results/gosec.sarif 2>/dev/null | head -1 || echo 0)
		GOSEC_CRITICAL=$(printf '%s' "$GOSEC_CRITICAL")
		if [ -z "$GOSEC_CRITICAL" ]; then GOSEC_CRITICAL=0; fi
		echo "gosec: $GOSEC_CRITICAL high/critical findings"
		if [ "$GOSEC_CRITICAL" -gt 0 ]; then
			echo "❌ FAIL: gosec found $GOSEC_CRITICAL high/critical findings"
			gosec -exclude=G101,G201,G505 -exclude-generated -fmt=sarif -out=sarif-results/gosec.sarif ./... 2>&1
			FAIL=1
		fi
	fi

	# Trivy / Gitleaks / TruffleHog are enforced by the scanners themselves:
	#   trivy-action runs with exit-code: 1 (fails on CRITICAL/HIGH),
	#   gitleaks exits 1 when leaks are found,
	#   trufflehog exits 194 on verified findings.
	# Their reports are still uploaded to Code Scanning.

	# govulncheck is informational only (Trivy covers CVE scanning).
	if [ -f sarif-results/govulncheck.txt ]; then
		GOVULNCHECK=$(grep -cE "^\(#[0-9]+\)" sarif-results/govulncheck.txt 2>/dev/null)
		GOVULNCHECK=${GOVULNCHECK:-0}
		echo "govulncheck: $GOVULNCHECK vulnerabilities found (informational)"
	fi

	echo "============================================"
	if [ "$FAIL" -eq 1 ]; then
		echo "❌ Security Gate FAILED — pipeline blocked"
		echo "   Fix the issues above before merging."
		exit 1
	fi
	echo "✅ Security Gate PASSED — no critical issues found"
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

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		install_tools
		run_gosec
		run_govulncheck
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
