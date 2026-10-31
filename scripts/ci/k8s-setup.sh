#!/bin/bash
set -euo pipefail

setup_local_path_provisioner() {
	./scripts/ssh-retry.sh scp configs/k8s/base/local-path-provisioner.yaml "${VPS_USER}@${VPS_HOST}:/tmp/local-path-provisioner.yaml"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
		set -euo pipefail
		echo 'Applying local-path-provisioner manifest...'
		sudo k3s kubectl apply -f /tmp/local-path-provisioner.yaml
		echo 'Restarting local-path-provisioner to clear any stuck state...'
		sudo k3s kubectl rollout restart deployment/local-path-provisioner -n local-path-storage || true
		# СТРОГО: провал rollout валит job (раньше был || true — маскировал сбой)
		sudo k3s kubectl rollout status deployment/local-path-provisioner -n local-path-storage --timeout=180s
		sudo k3s kubectl wait --for=condition=Ready pod -l app=local-path-provisioner -n local-path-storage --timeout=120s
		echo 'Verifying StorageClass local-path exists and is ready...'
		for i in {1..30}; do
			if sudo k3s kubectl get storageclass local-path &>/dev/null; then
				echo '✅ StorageClass found'
				break
			fi
			if [[ "$i" -eq 30 ]]; then
				echo '❌ StorageClass local-path not found'
				exit 1
			fi
			echo "Attempt $i/30: waiting for StorageClass..."
			sleep 2
		done
		if ! sudo k3s kubectl get storageclass local-path -o jsonpath='{.metadata.annotations.storageclass\.kubernetes\.io/is-default-class}' | grep -q true; then
			echo '⚠️ StorageClass local-path is not default, patching...'
			sudo k3s kubectl patch storageclass local-path -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
		fi
		echo '✅ local-path-provisioner and StorageClass are ready'
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
		echo '✅ ClusterIssuers created'

		echo '✅ Certificate configuration complete'
EOF
}

install_ingress_nginx() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p /tmp/ingress-nginx"
	./scripts/ssh-retry.sh scp -r configs/k8s/base/ingress-nginx/* "${VPS_USER}@${VPS_HOST}:/tmp/ingress-nginx/"

	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
		set -euo pipefail
		echo '-> Installing ingress-nginx controller...'
		sudo k3s kubectl apply -f /tmp/ingress-nginx/

		echo '-> Waiting for ingress-nginx to be ready...'
		sudo k3s kubectl wait --namespace ingress-nginx \
			--for=condition=ready pod \
			--selector=app.kubernetes.io/name=ingress-nginx \
			--timeout=180s || {
			echo '❌ ingress-nginx not ready'
			sudo k3s kubectl -n ingress-nginx get pods,deploy,svc
			exit 1
		}

		echo '✅ ingress-nginx is ready'
		sudo k3s kubectl -n ingress-nginx get pods,deploy,svc
EOF
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
