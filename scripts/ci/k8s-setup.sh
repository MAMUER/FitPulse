#!/bin/bash
set -euo pipefail
setup_local_path_provisioner() {
	./scripts/ssh-retry.sh scp configs/k8s/base/local-path-provisioner.yaml "${VPS_USER}@${VPS_HOST}:/tmp/local-path-provisioner.yaml"
	./scripts/ssh-retry.sh scp configs/k8s/scripts/fix-local-path.sh "${VPS_USER}@${VPS_HOST}:/tmp/fix-local-path.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo 'Applying local-path-provisioner manifest and running strict smoke test...'
sudo chmod +x /tmp/fix-local-path.sh
sudo /tmp/fix-local-path.sh
echo '✅ local-path-provisioner is ready and PVC provisioning works'
EOF
}
fix_local_path_rbac() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/fix-local-path.sh "${VPS_USER}@${VPS_HOST}:/tmp/fix-local-path.sh"
	./scripts/ssh-retry.sh scp configs/k8s/base/local-path-provisioner.yaml "${VPS_USER}@${VPS_HOST}:/tmp/local-path-provisioner.yaml"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/fix-local-path.sh && sudo /tmp/fix-local-path.sh"
}
install_cert_manager() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p /tmp/cert-manager"
	./scripts/ssh-retry.sh scp \
		configs/k8s/base/cert-manager/cluster-issuer-staging.yaml \
		configs/k8s/base/cert-manager/cluster-issuer.yaml \
		configs/k8s/base/cert-manager/grpc-selfsigned-ca.yaml \
		configs/k8s/base/cert-manager/rabbitmq-cert.yaml \
		"${VPS_USER}@${VPS_HOST}:/tmp/cert-manager/"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo '-> Installing cert-manager CRDs and controller...'
curl -fsSL https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml | sudo k3s kubectl apply -f -
echo '-> Waiting for cert-manager pods to be ready...'
sudo k3s kubectl wait --namespace cert-manager --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager --timeout=120s
echo '✅ cert-manager installed successfully'
echo '-> Creating Let'\''s Encrypt ClusterIssuers...'
sudo k3s kubectl apply -f /tmp/cert-manager/cluster-issuer-staging.yaml
sudo k3s kubectl apply -f /tmp/cert-manager/cluster-issuer.yaml
sudo k3s kubectl apply -f /tmp/cert-manager/grpc-selfsigned-ca.yaml
sudo k3s kubectl apply -f /tmp/cert-manager/rabbitmq-cert.yaml
echo '✅ ClusterIssuers created'
echo '✅ Certificate configuration complete'
EOF
}
install_ingress_nginx() {
	# Apply the same overlay the production kustomization uses: base controller
	# + namespace-scoped TLS Role + --watch-namespace patch. Applying the bare
	# base used to start the controller with cluster-wide secret watches, which
	# RBAC denies (secrets are intentionally not granted cluster-wide), so it
	# crash-looped on /healthz 500 until the later overlay apply.
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p /tmp/ingress-nginx/base/ingress-nginx" || return 0
	./scripts/ssh-retry.sh scp -r configs/k8s/base/ingress-nginx/* "${VPS_USER}@${VPS_HOST}:/tmp/ingress-nginx/base/ingress-nginx/" || return 0
	./scripts/ssh-retry.sh scp \
		configs/k8s/overlays/production/ingress-nginx-tls-role.yaml \
		configs/k8s/overlays/production/ingress-nginx-watch-namespace-patch.yaml \
		"${VPS_USER}@${VPS_HOST}:/tmp/ingress-nginx/" || return 0

	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF' || true
set -euo pipefail
cat > /tmp/ingress-nginx/kustomization.yaml <<'YAML'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - base/ingress-nginx
  - ingress-nginx-tls-role.yaml
patches:
  - path: ingress-nginx-watch-namespace-patch.yaml
    target:
      kind: Deployment
      name: ingress-nginx-controller
      namespace: ingress-nginx
YAML
echo '-> Installing ingress-nginx controller...'
sudo k3s kubectl apply -k /tmp/ingress-nginx/ || true

echo '-> Waiting for ingress-nginx to be ready...'
sudo k3s kubectl wait --namespace ingress-nginx \
	--for=condition=ready pod \
	--selector=app.kubernetes.io/name=ingress-nginx \
	--timeout=120s || {
	echo 'ingress-nginx not ready (image may not be available). Skipping.'
	sudo k3s kubectl -n ingress-nginx get deploy,rs,pods,svc -o wide || true
	sudo k3s kubectl -n ingress-nginx describe deployment ingress-nginx-controller || true
	while IFS= read -r pod; do
		[ -n "$pod" ] || continue
		phase=$(sudo k3s kubectl -n ingress-nginx get pod "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || echo Unknown)
		ready=$(sudo k3s kubectl -n ingress-nginx get pod "$pod" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo Unknown)
		[ "$ready" = True ] && [ "$phase" != Failed ] && continue
		echo "=== Pod diagnostics: ingress-nginx/$pod (phase=$phase, ready=$ready) ==="
		sudo k3s kubectl -n ingress-nginx describe pod "$pod" || true
		containers=$(sudo k3s kubectl -n ingress-nginx get pod "$pod" -o jsonpath='{range .spec.initContainers[*]}{.name}{"\n"}{end}{range .spec.containers[*]}{.name}{"\n"}{end}' 2>/dev/null || true)
		restarts=$(sudo k3s kubectl -n ingress-nginx get pod "$pod" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}={.restartCount}{"\n"}{end}{range .status.containerStatuses[*]}{.name}={.restartCount}{"\n"}{end}' 2>/dev/null || true)
		states=$(sudo k3s kubectl -n ingress-nginx get pod "$pod" -o jsonpath='{range .status.initContainerStatuses[*]}init/{.name}: waiting={.state.waiting.reason} terminated={.state.terminated.reason} last={.lastState.terminated.reason} exit={.lastState.terminated.exitCode} message={.lastState.terminated.message} restarts={.restartCount}{"\n"}{end}{range .status.containerStatuses[*]}{.name}: waiting={.state.waiting.reason} terminated={.state.terminated.reason} last={.lastState.terminated.reason} exit={.lastState.terminated.exitCode} message={.lastState.terminated.message} restarts={.restartCount}{"\n"}{end}' 2>/dev/null || true)
		echo '--- Container states ---'
		[ -n "$states" ] && printf '%s\n' "$states" || echo 'No container status has been reported yet'
		while IFS= read -r container; do
			[ -n "$container" ] || continue
			echo "--- Current logs: $container ---"
			sudo k3s kubectl -n ingress-nginx logs "$pod" -c "$container" --timestamps --tail=150 2>&1 || true
			restart_count=$(printf '%s\n' "$restarts" | awk -F= -v name="$container" '$1 == name {print $2; exit}')
			if [ -n "$restart_count" ] && [ "$restart_count" -gt 0 ] 2>/dev/null; then
				echo "--- Previous logs: $container (restart count: $restart_count) ---"
				sudo k3s kubectl -n ingress-nginx logs "$pod" -c "$container" --previous --timestamps --tail=150 2>&1 || true
			fi
		done <<<"$containers"
		echo "--- Pod events ---"
		sudo k3s kubectl -n ingress-nginx get events --field-selector "involvedObject.name=$pod" --sort-by=.lastTimestamp || true
	done < <(sudo k3s kubectl -n ingress-nginx get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)
	echo '=== Recent ingress-nginx events ==='
	sudo k3s kubectl -n ingress-nginx get events --sort-by=.lastTimestamp | tail -50 || true
	exit 0
}
echo 'ingress-nginx is ready'
sudo k3s kubectl -n ingress-nginx get pods,deploy,svc || true
EOF
	echo 'ingress-nginx step completed (non-fatal)'
}

cleanup_traefik() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo '-> Cleaning up Traefik from kube-system...'
sudo k3s kubectl -n kube-system delete deployment traefik --ignore-not-found=true || true
sudo k3s kubectl -n kube-system delete svc traefik --ignore-not-found=true || true
sudo k3s kubectl -n kube-system get pods,svc,deploy | grep -i traefik || echo 'Traefik already removed or not found'
echo '✅ Traefik cleanup complete (non-fatal)'
EOF
}
main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		setup_local_path_provisioner
		fix_local_path_rbac
		install_cert_manager
		install_ingress_nginx
		cleanup_traefik
		;;
	setup_local_path_provisioner) setup_local_path_provisioner ;;
	fix_local_path_rbac) fix_local_path_rbac ;;
	install_cert_manager) install_cert_manager ;;
	install_ingress_nginx) install_ingress_nginx ;;
	cleanup_traefik) cleanup_traefik ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}
main "$@"
