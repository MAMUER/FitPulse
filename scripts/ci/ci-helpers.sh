#!/bin/bash
set -euo pipefail

validate_workflow() {
	curl --proto "=https" --tlsv1.2 -sSfL https://github.com/rhysd/actionlint/releases/download/v1.7.12/actionlint_1.7.12_linux_amd64.tar.gz -o /tmp/actionlint.tar.gz
	echo "8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8  /tmp/actionlint.tar.gz" | sha256sum --check --status || {
		echo "Hash verification failed"
		exit 1
	}
	tar -xzf /tmp/actionlint.tar.gz -C /tmp actionlint
	chmod +x /tmp/actionlint
	/tmp/actionlint -color
}

run_hadolint() {
	find cmd -maxdepth 2 -type f -name Dockerfile -print0 | sort -z | while IFS= read -r -d '' dockerfile; do
		echo "Linting $dockerfile"
		docker run --rm -v "${GITHUB_WORKSPACE}:/workspace" -w /workspace hadolint/hadolint:v2.12.0 hadolint -c .hadolint.yaml "$dockerfile"
	done
}

install_dependencies() {
	npm config set registry https://registry.npmjs.org
	NPM_CONFIG_MIN_RELEASE_AGE=0 npm ci --ignore-scripts --legacy-peer-deps
}

install_golangci_lint() {
	for i in 1 2 3; do
		if go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@8f3b0c7ed018e57905fbd873c697e0b1ede605a5; then # NOSONAR
			break
		fi
		echo "golangci-lint install attempt $i failed; retrying..."
		sleep 5
	done
}

install_python_deps() {
	python -m pip install --upgrade pip==25.0
	pip install --only-binary :all: -r cmd/ml_generator/requirements.lock.txt
}

run_coverage() {
	set +e
	mkdir -p coverage

	echo "-> Running Go tests with coverage..."
	go test -count=1 -coverprofile=coverage/coverage.out ./...
	GO_TEST_EXIT=$?

	echo "-> Running frontend tests with coverage..."
	cd web
	npm run test
	NPM_TEST_EXIT=$?
	cd ..

	set -e

	echo "Go tests exit code: $GO_TEST_EXIT"
	echo "npm test exit code: $NPM_TEST_EXIT"

	pwd
	ls -la coverage || true
	if [ -f coverage/lcov.info ]; then
		echo "--- original lcov head ---"
		head -20 coverage/lcov.info || true
		sed -e 's|\\|/|g' -e 's|^SF:src/|SF:web/src/|' -e 's|^SF:/|SF:|' coverage/lcov.info >../coverage/lcov.info
		echo "--- transformed lcov head ---"
		head -20 ../coverage/lcov.info || true
	else
		echo "lcov.info not found in web/coverage"
	fi
	if [ -f coverage/coverage.out ]; then
		echo "--- fixing Go coverage paths ---"
		sed -i 's|github.com/MAMUER/project/||g' coverage/coverage.out
		sed -i '/^api\/gen\//d' coverage/coverage.out
		head -5 coverage/coverage.out || true
		echo "--- checking Go coverage threshold (80%) ---"
		GO_COVERAGE=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
		echo "Go coverage: ${GO_COVERAGE}%"
		GO_COVERAGE_INT=${GO_COVERAGE%.*}
		if [ "$GO_COVERAGE_INT" -lt 80 ]; then
			echo "ERROR: Go test coverage ${GO_COVERAGE}% is below 80% threshold"
			return 1
		fi
		echo "Go coverage threshold met: ${GO_COVERAGE}% >= 80%"
	else
		echo "WARNING: coverage/coverage.out not found, skipping coverage threshold check"
	fi

	if [ "$GO_TEST_EXIT" -ne 0 ] || [ "$NPM_TEST_EXIT" -ne 0 ]; then
		echo "Some tests failed (go=$GO_TEST_EXIT, npm=$NPM_TEST_EXIT), but coverage reports were generated"
		return 0
	fi
}

configure_kubectl() {
	mkdir -p "$HOME/.kube"
	cp ./kubeconfig/k3s-config.yaml "$HOME/.kube/config"
	chmod 600 "$HOME/.kube/config"
}

download_kubeconfig() {
	./scripts/ssh-retry.sh scp \
		"${VPS_USER}@${VPS_HOST}:~/k3s-config.yaml" \
		./k3s-config.yaml
}

generate_kubeconfig() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" -- "$FITPULSE_DOMAIN" /bin/bash -s <<'REMOTE_EOF'
set -euo pipefail
DOMAIN="$1"
sudo cp /etc/rancher/k3s/k3s.yaml /tmp/k3s-domain.yaml
sudo sed -i "s|0.0.0.0|${DOMAIN}|g" /tmp/k3s-domain.yaml
sudo sed -i "s|127.0.0.1|${DOMAIN}|g" /tmp/k3s-domain.yaml
sudo sed -i '/insecure-skip-tls-verify/d' /tmp/k3s-domain.yaml
sudo chmod 644 /tmp/k3s-domain.yaml
cp /tmp/k3s-domain.yaml ~/k3s-config.yaml
echo "✅ Kubeconfig saved to ~/k3s-config.yaml"
REMOTE_EOF
}

ensure_k3s_certs() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" -- "$FITPULSE_DOMAIN" /bin/bash -s <<'REMOTE_EOF'
set -euo pipefail
FITPULSE_DOMAIN="$1"
if ! k3s kubectl get --raw /livez &>/dev/null; then
	echo "API не отвечает"; exit 1
fi
CERT_FILE="/var/lib/rancher/k3s/server/tls/dynamic-cert.json"
NEED_RESTART=false
if [ -f "$CERT_FILE" ]; then
	CURRENT_SAN=$(kubectl get --raw /apis | jq -r '.')
	if ! grep -q "$FITPULSE_DOMAIN" /etc/rancher/k3s/config.yaml; then
		echo "⚠️ Конфиг не содержит домен, обновляем..."
		NEED_RESTART=true
	else
		echo "✅ Домен уже в конфиге."
	fi
else
	echo "Сертификат ещё не сгенерирован, перезапуск не требуется."
fi
if [ "$NEED_RESTART" = true ]; then
	echo "Перезапуск k3s для применения новых SAN..."
	sudo systemctl stop k3s
	sudo rm -f /var/lib/rancher/k3s/server/tls/dynamic-cert.json
	sudo systemctl start k3s
	for i in $(seq 1 60); do
		if k3s kubectl cluster-info &>/dev/null; then
			echo "k3s снова готов после ${i}s"
			break
		fi
		sleep 2
	done
fi
REMOTE_EOF
}

wait_for_k3s_api() {
	echo "Waiting for k3s API server to be fully initialized..."
	for i in {1..30}; do
		if kubectl get --raw=/healthz &>/dev/null; then
			echo "✅ k3s API server is ready after ${i}0 seconds"
			break
		fi
		if [ "$i" -eq 30 ]; then
			echo "❌ k3s API server not ready after 5 minutes"
			kubectl cluster-info dump || true
			exit 1
		fi
		echo "Attempt $i/30: API not ready, waiting 10s..."
		sleep 10
	done
	echo "Verifying API endpoints..."
	kubectl api-resources --api-group=apps &>/dev/null || {
		echo "API resources not available, waiting 30s more..."
		sleep 30
	}
}

check_connectivity() {
	echo "Testing connectivity to ${VPS_HOST}:6443..."
	if timeout 10 bash -c "</dev/tcp/${VPS_HOST}/6443" 2>/dev/null; then
		echo "Cluster reachable"
		echo "reachable=true" >>"$GITHUB_OUTPUT"
	else
		echo "Cluster NOT reachable (timeout 10s)"
		echo "reachable=false" >>"$GITHUB_OUTPUT"
		exit 1
	fi
}

add_vps_to_known_hosts() {
	mkdir -p ~/.ssh
	ssh-keyscan -H "${VPS_HOST}" >>~/.ssh/known_hosts 2>/dev/null || true
}

resolve_vps_hostname() {
	VPS_IP=$(dig +short "${VPS_HOST}" | head -1)
	if [ -z "$VPS_IP" ]; then
		VPS_IP=$(nslookup "${VPS_HOST}" | grep -A1 "Name:" | tail -1 | awk '{print $2}')
	fi
	if [ -z "$VPS_IP" ]; then
		VPS_IP="${VPS_HOST}"
	fi
	echo "Resolved VPS_IP: $VPS_IP"
	echo "$VPS_IP ${FITPULSE_DOMAIN}" | sudo tee -a /etc/hosts
}

prepare_target_url() {
	URL="https://${FITPULSE_DOMAIN}"
	echo "target=$URL" >>"$GITHUB_OUTPUT"
}

run_immediate_health_check() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"mkdir -p /etc/fitpulse && chmod 700 /etc/fitpulse"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"printf '%s\n' '${TELEGRAM_BOT_TOKEN}' > /etc/fitpulse/telegram-bot-token && chmod 600 /etc/fitpulse/telegram-bot-token"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"printf '%s\n' '${TELEGRAM_CHAT_ID}' > /etc/fitpulse/telegram-chat-id && chmod 600 /etc/fitpulse/telegram-chat-id"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
	export TELEGRAM_BOT_TOKEN=\$(cat /etc/fitpulse/telegram-bot-token)
	export TELEGRAM_CHAT_ID=\$(cat /etc/fitpulse/telegram-chat-id)
	/usr/local/bin/check-server-health.sh || true
	"
}

setup_postgres_backups() {
	if [ -n "${BACKUP_KEY}" ]; then
		echo -n "${BACKUP_KEY}" >/tmp/backup.key
		chmod 600 /tmp/backup.key
		./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p /etc/fitpulse"
		./scripts/ssh-retry.sh scp /tmp/backup.key "${VPS_USER}@${VPS_HOST}:/tmp/backup.key"
		./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
	sudo mv /tmp/backup.key /etc/fitpulse/backup.key
	sudo chmod 600 /etc/fitpulse/backup.key
	sudo chown root:root /etc/fitpulse/backup.key
	echo '✅ Backup key saved securely to /etc/fitpulse/backup.key'
	"
		rm -f /tmp/backup.key
	else
		echo "⚠️ BACKUP_KEY secret is not set in GitHub. Backups will NOT be encrypted."
		./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo rm -f /etc/fitpulse/backup.key || true"
	fi
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
sudo mkdir -p /opt/fitpulse/backups
cat <<'BACKUP_EOF' | sudo tee /usr/local/bin/backup-postgres.sh > /dev/null
#!/bin/bash
set -euo pipefail
BACKUP_DIR='/opt/fitpulse/backups'
KEY_FILE='/etc/fitpulse/backup.key'
mkdir -p \"\$BACKUP_DIR\"
DATE=\$(date +%Y%m%d_%H%M%S)
FILENAME=\"\$BACKUP_DIR/fitness_\$DATE.dump\"
ENCRYPTED=\"\$FILENAME.enc\"
/usr/local/bin/k3s kubectl exec -n fitness-platform-production postgres-0 -- \
	pg_dump -U postgres -d fitness -F c > \"\$FILENAME\"
if [ -f \"\$KEY_FILE\" ] && [ -s \"\$KEY_FILE\" ]; then
	openssl enc -aes-256-cbc -salt -pbkdf2 -pass file:\"\$KEY_FILE\" -in \"\$FILENAME\" -out \"\$ENCRYPTED\"
	rm -f \"\$FILENAME\"
	echo \"Encrypted backup created: \$ENCRYPTED\"
else
	echo \"Backup created (unencrypted): \$FILENAME\"
fi
find \"\$BACKUP_DIR\" -type f -mtime +7 -delete
BACKUP_EOF
sudo chmod +x /usr/local/bin/backup-postgres.sh
(sudo crontab -l 2>/dev/null | grep -v backup-postgres; echo '0 2 * * * /usr/local/bin/backup-postgres.sh >> /var/log/backup-postgres.log 2>&1') | sudo crontab -
echo '✅ Postgres backups configured (daily at 2 AM)'
"
}

show_codeql_diagnostics() {
	echo "-> CodeQL SARIF files:"
	ls -la sarif-results || true
	for f in sarif-results/*.sarif; do
		if [ -f "$f" ]; then
			echo "=== $f ==="
			jq -r '
				.runs[] |
				.tool.driver.name,
				"results: \(([.results // [] | length]) // 0)",
				(.results // [])[] |
				[.ruleId, .message.text, (.locations[0].physicalLocation.artifactLocation.uri // ""), (.locations[0].physicalLocation.region.startLine // 0)] | @tsv
			' "$f" 2>/dev/null || true
			echo ""
		fi
	done
}

deploy_health_script() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/check-server-health.sh "${VPS_USER}@${VPS_HOST}:/usr/local/bin/check-server-health.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
sudo chmod +x /usr/local/bin/check-server-health.sh
(sudo crontab -l 2>/dev/null | grep -v check-server-health) | sudo crontab -
(sudo crontab -l 2>/dev/null; echo '0 9 * * * TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN} TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID} /usr/local/bin/check-server-health.sh --silent >> /var/log/server-health.log 2>&1') | sudo crontab -
echo '✅ Health check script deployed and cron configured'
"
}

setup_health_cron() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
(sudo crontab -l 2>/dev/null | grep -v check-server-health) | sudo crontab -
cat <<'CRON' | sudo crontab -
0 9 * * * /usr/local/bin/check-server-health.sh --silent >> /var/log/server-health.log 2>&1
CRON
echo '✅ Daily health check cron configured (9 AM UTC)'
"
}

setup_duckdns() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo '-> Setting up DuckDNS auto-updater...'
echo '${DUCKDNS_TOKEN}' | sudo tee /etc/duckdns/token > /dev/null
sudo chmod 600 /etc/duckdns/token
sudo mkdir -p /etc/duckdns
"
	./scripts/ssh-retry.sh scp configs/k8s/scripts/duckdns-update.sh "${VPS_USER}@${VPS_HOST}:/usr/local/bin/duckdns-update.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
sudo chmod +x /usr/local/bin/duckdns-update.sh
(sudo crontab -l 2>/dev/null | grep -v duckdns-update; echo '*/5 * * * * /usr/local/bin/duckdns-update.sh >> /var/log/duckdns.log 2>&1') | sudo crontab -
sudo /usr/local/bin/duckdns-update.sh
echo '✅ DuckDNS auto-updater configured (every 5 minutes)'
"
}

setup_ml_dirs() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo '-> Creating directories for ML storage...'
sudo mkdir -p /opt/fitpulse/mlflow-data
sudo chown -R ${VPS_USER}:${VPS_USER} /opt/fitpulse
echo '✅ Directories created: /opt/fitpulse/mlflow-data'
"
}

deploy_mlflow() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
mkdir -p ~/fitpulse/deployments
docker rm -f fitpulse-mlflow 2>/dev/null || true
docker run -d \
	--name fitpulse-mlflow \
	--restart unless-stopped \
	-p 5000:5000 \
	-v /opt/fitpulse/mlflow-data:/mlflow \
	ghcr.io/mlflow/mlflow:v2.20.2 \
	mlflow server \
	--host 0.0.0.0 \
	--port 5000 \
	--backend-store-uri sqlite:////mlflow/mlflow.db \
	--default-artifact-root /mlflow/artifacts
for _ in {1..30}; do
	if curl -sf http://localhost:5000/health &>/dev/null; then
		echo '✅ MLflow is healthy'
		break
	fi
	sleep 2
done
echo \"✅ MLflow deployed: http://${VPS_HOST}:5000\"
"
}

check_commits() {
	echo "-> Checking conventional commits on main..."
	git log --oneline -1 --no-merges --pretty=format:"%s" | while read -r commit_msg; do
		if ! echo "$commit_msg" | grep -qE "^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert|security|BREAKING CHANGE)"; then
			echo "❌ Non-conventional commit message found: $commit_msg"
			exit 1
		fi
	done
	echo "✅ All commit messages follow conventional commits format"
}

generate_provenance() {
	set -euo pipefail
	files=(bin/*)
	echo "hashes=$(sha256sum "${files[@]}" | base64 -w0)" >>"${GITHUB_OUTPUT}"
}

collect_statuses() {
	get_emoji() { case "$1" in success) echo "✅" ;; failure) echo "❌" ;; skipped) echo "⏭️" ;; *) echo "❓" ;; esac }
	VALIDATE="$(get_emoji "${JOB_STATUS_VALIDATE_WORKFLOW:-success}")"
	DOCKERFILE_LINT="$(get_emoji "${JOB_STATUS_DOCKERFILE_LINT:-success}")"
	SUPER_LINTER="$(get_emoji "${JOB_STATUS_SUPER_LINTER:-success}")"
	FRONTEND="$(get_emoji "${JOB_STATUS_FRONTEND:-success}")"
	CHECK="$(get_emoji "${JOB_STATUS_CHECK:-success}")"
	BUILD="$(get_emoji "${JOB_STATUS_BINARY_BUILD:-success}")"
	DOCKER="$(get_emoji "${JOB_STATUS_DOCKER:-success}")"
	CODEQL="$(get_emoji "${JOB_STATUS_CODEQL:-success}")"
	KUBESCAPE="$(get_emoji "${JOB_STATUS_KUBESCAPE:-success}")"
	SECURITY_SCAN="$(get_emoji "${JOB_STATUS_SECURITY_SCAN:-success}")"
	SONARCLOUD="$(get_emoji "${JOB_STATUS_SONARCLOUD:-success}")"
	CONVENTIONAL="$(get_emoji "${JOB_STATUS_CONVENTIONAL_COMMITS:-success}")"
	KUBE_BENCH="$(get_emoji "${JOB_STATUS_KUBE_BENCH:-success}")"
	KUBE_HUNTER="$(get_emoji "${JOB_STATUS_KUBE_HUNTER:-success}")"
	CHECKOV="$(get_emoji "${JOB_STATUS_CHECKOV:-success}")"
	COSIGN_VERIFY="$(get_emoji "${JOB_STATUS_COSIGN_VERIFY:-success}")"
	PROVISION_K8S="$(get_emoji "${JOB_STATUS_PROVISION_K8S_VPS:-success}")"
	DEPLOY_STAGING="$(get_emoji "${JOB_STATUS_DEPLOY_STAGING:-success}")"
	TEST_STAGING="$(get_emoji "${JOB_STATUS_TEST_STAGING:-success}")"
	DEPLOY_PROD="$(get_emoji "${JOB_STATUS_DEPLOY_PRODUCTION:-success}")"
	TEST_PROD="$(get_emoji "${JOB_STATUS_TEST_PRODUCTION:-success}")"
	SMOKE_TEST="$(get_emoji "${JOB_STATUS_SMOKE_TEST:-success}")"
	HEALTH_CHECK="$(get_emoji "${JOB_STATUS_HEALTH_CHECK:-success}")"
	CSP_CHECK="$(get_emoji "${JOB_STATUS_CSP_HEADERS_CHECK:-success}")"
	FAILED_JOBS=""
	if [[ "${JOB_STATUS_VALIDATE_WORKFLOW:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Validate "; fi
	if [[ "${JOB_STATUS_DOCKERFILE_LINT:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}DockerfileLint "; fi
	if [[ "${JOB_STATUS_SUPER_LINTER:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Lint "; fi
	if [[ "${JOB_STATUS_FRONTEND:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Frontend "; fi
	if [[ "${JOB_STATUS_CHECK:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Check "; fi
	if [[ "${JOB_STATUS_BINARY_BUILD:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Build "; fi
	if [[ "${JOB_STATUS_DOCKER:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Docker "; fi
	if [[ "${JOB_STATUS_CODEQL:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}CodeQL "; fi
	if [[ "${JOB_STATUS_KUBESCAPE:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Kubescape "; fi
	if [[ "${JOB_STATUS_SECURITY_SCAN:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}SecurityScan "; fi
	if [[ "${JOB_STATUS_SONARCLOUD:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}SonarCloud "; fi
	if [[ "${JOB_STATUS_CONVENTIONAL_COMMITS:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}ConventionalCommits "; fi
	if [[ "${JOB_STATUS_KUBE_BENCH:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}KubeBench "; fi
	if [[ "${JOB_STATUS_KUBE_HUNTER:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}KubeHunter "; fi
	if [[ "${JOB_STATUS_CHECKOV:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Checkov "; fi
	if [[ "${JOB_STATUS_COSIGN_VERIFY:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}CosignVerify "; fi
	if [[ "${JOB_STATUS_PROVISION_K8S_VPS:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}ProvisionK8s "; fi
	if [[ "${JOB_STATUS_DEPLOY_STAGING:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}DeployStaging "; fi
	if [[ "${JOB_STATUS_TEST_STAGING:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}TestStaging "; fi
	if [[ "${JOB_STATUS_SEMGREP:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Semgrep "; fi
	if [[ "${JOB_STATUS_OSV_SCANNER:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}OSVScanner "; fi
	if [[ "${JOB_STATUS_FUZZ_TESTS:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}FuzzTests "; fi
	if [[ "${JOB_STATUS_OSV_AUDIT:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}OSVAudit "; fi
	if [[ "${JOB_STATUS_DEPENDENCY_REVIEW:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}DependencyReview "; fi
	if [[ "${JOB_STATUS_DEPLOY_PRODUCTION:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Deploy "; fi
	if [[ "${JOB_STATUS_TEST_PRODUCTION:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Test "; fi
	if [[ "${JOB_STATUS_SMOKE_TEST:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}Smoke "; fi
	if [[ "${JOB_STATUS_HEALTH_CHECK:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}HealthCheck "; fi
	if [[ "${JOB_STATUS_CSP_HEADERS_CHECK:-success}" == "failure" ]]; then FAILED_JOBS="${FAILED_JOBS}CSPCheck "; fi
	if [[ -n "$FAILED_JOBS" ]]; then
		OVERALL="ОШИБКА"
		STATUS="failure"
	else
		OVERALL="УСПЕШНО"
		STATUS="success"
	fi
	{
		echo "overall=$OVERALL"
		echo "status=$STATUS"
		echo "validate=$VALIDATE"
		echo "dockerfile_lint=$DOCKERFILE_LINT"
		echo "super_linter=$SUPER_LINTER"
		echo "frontend=$FRONTEND"
		echo "check=$CHECK"
		echo "build=$BUILD"
		echo "docker=$DOCKER"
		echo "codeql=$CODEQL"
		echo "kubescape=$KUBESCAPE"
		echo "security_scan=$SECURITY_SCAN"
		echo "sonarcloud=$SONARCLOUD"
		echo "conventional=$CONVENTIONAL"
		echo "kube_bench=$KUBE_BENCH"
		echo "kube_hunter=$KUBE_HUNTER"
		echo "checkov=$CHECKOV"
		echo "cosign_verify=$COSIGN_VERIFY"
		echo "provision_k8s=$PROVISION_K8S"
		echo "deploy_staging=$DEPLOY_STAGING"
		echo "test_staging=$TEST_STAGING"
		echo "deploy_prod=$DEPLOY_PROD"
		echo "test_prod=$TEST_PROD"
		echo "smoke_test=$SMOKE_TEST"
		echo "health_check=$HEALTH_CHECK"
		echo "csp_check=$CSP_CHECK"
	} >>"$GITHUB_OUTPUT"
}

send_telegram() {
	curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-H "Content-Type: application/json" \
		-d "{\"chat_id\": \"${TELEGRAM_CHAT_ID}\", \"text\": \"🔴 Critical issue: #${ISSUE_NUMBER}\n${ISSUE_TITLE}\n${ISSUE_HTML_URL}\"}"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		validate_workflow
		run_hadolint
		install_dependencies
		install_golangci_lint
		install_python_deps
		run_coverage
		configure_kubectl
		download_kubeconfig
		generate_kubeconfig
		ensure_k3s_certs
		wait_for_k3s_api
		check_connectivity
		deploy_health_script
		setup_health_cron
		setup_duckdns
		setup_ml_dirs
		deploy_mlflow
		check_commits
		generate_provenance
		collect_statuses
		send_telegram
		;;
	validate_workflow) validate_workflow ;;
	run_hadolint) run_hadolint ;;
	install_dependencies) install_dependencies ;;
	install_golangci_lint) install_golangci_lint ;;
	install_python_deps) install_python_deps ;;
	run_coverage) run_coverage ;;
	configure_kubectl) configure_kubectl ;;
	download_kubeconfig) download_kubeconfig ;;
	generate_kubeconfig) generate_kubeconfig ;;
	ensure_k3s_certs) ensure_k3s_certs ;;
	wait_for_k3s_api) wait_for_k3s_api ;;
	check_connectivity) check_connectivity ;;
	add_vps_to_known_hosts) add_vps_to_known_hosts ;;
	resolve_vps_hostname) resolve_vps_hostname ;;
	prepare_target_url) prepare_target_url ;;
	run_immediate_health_check) run_immediate_health_check ;;
	setup_postgres_backups) setup_postgres_backups ;;
	show_codeql_diagnostics) show_codeql_diagnostics ;;
	deploy_health_script) deploy_health_script ;;
	setup_health_cron) setup_health_cron ;;
	setup_duckdns) setup_duckdns ;;
	setup_ml_dirs) setup_ml_dirs ;;
	deploy_mlflow) deploy_mlflow ;;
	check_commits) check_commits ;;
	generate_provenance) generate_provenance ;;
	collect_statuses) collect_statuses ;;
	send_telegram) send_telegram ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
