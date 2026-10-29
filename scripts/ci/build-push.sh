#!/bin/bash
set -euo pipefail

build_go_services() {
	# gateway and device-aggregator keep their own Dockerfiles (web-builder stage
	# and Debian base respectively); the rest share Dockerfile.service.
	SHA=$(git rev-parse --short HEAD)
	for svc in user-service biometric-service training-service gateway device-aggregator; do
		echo "Building $svc..."
		dockerfile="Dockerfile.service"
		build_args=()
		case "$svc" in
		user-service) build_args=(--build-arg SERVICE=user-service --build-arg PORTS=50051) ;;
		biometric-service) build_args=(--build-arg SERVICE=biometric-service --build-arg PORTS=50052) ;;
		training-service) build_args=(--build-arg SERVICE=training-service --build-arg PORTS=50053) ;;
		gateway) dockerfile="cmd/gateway/Dockerfile" ;;
		device-aggregator) dockerfile="cmd/device-aggregator/Dockerfile" ;;
		*)
			echo "No build config for $svc"
			exit 1
			;;
		esac
		docker buildx build \
			--build-arg GOPROXY=https://proxy.golang.org,direct \
			--cache-from=type=gha \
			--cache-to=type=gha,mode=max \
			-t "$REGISTRY/mamuer/project/$svc:$SHA" \
			-t "$REGISTRY/mamuer/project/$svc:latest" \
			-f "$dockerfile" . \
			"${build_args[@]}" \
			--push
	done
}

build_ml_images() {
	if [[ -f "models/scaler.pkl" ]] && { [[ -f "models/classifier.keras" ]] || ls models/classifier_*.keras >/dev/null 2>&1; }; then
		SHA=$(git rev-parse --short HEAD)
		docker build -t "$REGISTRY/mamuer/project/classifier:$SHA" \
			-t "$REGISTRY/mamuer/project/classifier:latest" \
			-f cmd/classifier/Dockerfile .
		docker push "$REGISTRY/mamuer/project/classifier:$SHA"
		docker push "$REGISTRY/mamuer/project/classifier:latest"
	fi
}

build_crs_updater() {
	docker buildx build \
		--cache-from=type=gha \
		--cache-to=type=gha,mode=max \
		-t "$REGISTRY/mamuer/project/crs-updater:v1" \
		-f "configs/k8s/base/jobs/Dockerfile" configs/k8s/base/jobs \
		--push
}

build_open_wearables() {
	echo "Cloning Open Wearables repo..."
	git clone --depth 1 https://github.com/the-momentum/open-wearables.git /tmp/open-wearables
	echo "Building Open Wearables backend..."
	SHA=$(git rev-parse --short HEAD)
	docker build -t ghcr.io/mamuer/project/open-wearables-backend:"$SHA" \
		-t ghcr.io/mamuer/project/open-wearables-backend:latest \
		-f /tmp/open-wearables/backend/Dockerfile /tmp/open-wearables/backend
	echo "Building Open Wearables frontend..."
	docker build -t ghcr.io/mamuer/project/open-wearables-frontend:"$SHA" \
		-t ghcr.io/mamuer/project/open-wearables-frontend:latest \
		-f /tmp/open-wearables/frontend/Dockerfile /tmp/open-wearables/frontend
	echo "Pushing Open Wearables images..."
	docker push ghcr.io/mamuer/project/open-wearables-backend:"$SHA"
	docker push ghcr.io/mamuer/project/open-wearables-backend:latest
	docker push ghcr.io/mamuer/project/open-wearables-frontend:"$SHA"
	docker push ghcr.io/mamuer/project/open-wearables-frontend:latest
	echo "Open Wearables images pushed"
}

build_go_binaries() {
	set -euo pipefail
	mkdir -p bin
	echo "-> Downloading Go modules sequentially to avoid race conditions..."
	go mod download
	echo "-> Building Go services in parallel..."
	GOMAXPROCS=$(nproc)
	export GOMAXPROCS
	# Discover every Go main package under cmd/ so new services are picked up
	# automatically (e.g. admin-cli, classifier, data-processor).
	services=()
	while IFS= read -r svc; do
		[[ -n "$svc" ]] && services+=("$svc")
	done < <(find cmd -mindepth 2 -maxdepth 2 -name main.go -printf '%h\n' | sed 's|^cmd/||' | sort)
	pids=()
	names=()
	for svc in "${services[@]}"; do
		echo "  Building $svc..."
		go build -ldflags="-s -w" -o "bin/$svc" ./cmd/"$svc" &
		pids+=($!)
		names+=("$svc")
	done
	for i in "${!pids[@]}"; do
		wait "${pids[$i]}" || {
			echo "❌ Build failed for ${names[$i]}"
			exit 1
		}
	done
	echo "✅ All binaries built successfully (${#services[@]} services)"
	ls -lh bin/
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		build_go_services
		build_ml_images
		build_crs_updater
		build_open_wearables
		build_go_binaries
		;;
	build_go_services) build_go_services ;;
	build_ml_images) build_ml_images ;;
	build_crs_updater) build_crs_updater ;;
	build_open_wearables) build_open_wearables ;;
	build_go_binaries) build_go_binaries ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
