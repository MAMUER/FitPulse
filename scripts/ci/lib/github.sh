#!/usr/bin/env bash
# Helpers for downloading GitHub release assets.
#
# GitHub intermittently answers requests to the public release endpoint
# (github.com/<repo>/releases/download/...) with HTTP 504. The same bytes are
# also served by the REST API asset endpoint (api.github.com), so these
# helpers try the public URL first and fall back to the API. Set GITHUB_TOKEN
# to raise the API rate limit; the token is optional.

if [[ -n "${__GITHUB_SH_LOADED:-}" ]]; then
	return 0
fi
__GITHUB_SH_LOADED=1

# Informational output goes to stderr: callers capture stdout.
github_log() {
	printf '%s\n' "$*" >&2
}

# Download a URL to a destination atomically. Each attempt uses a fresh .part
# file, so a truncated download is never mistaken for a complete artifact.
# Extra arguments are passed to curl verbatim (e.g. -H "Accept: ...").
github_download_url() {
	local dest="$1" url="$2"
	shift 2
	local attempts="${GITHUB_DOWNLOAD_ATTEMPTS:-5}"
	local attempt
	rm -f "$dest.part"
	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if curl --proto '=https' --tlsv1.2 -fsSL --connect-timeout 30 \
			--retry 1 --retry-delay 3 --retry-all-errors \
			"$@" -o "$dest.part" "$url"; then
			mv -f "$dest.part" "$dest"
			return 0
		fi
		github_log "Download failed (attempt ${attempt}/${attempts}): ${url}"
		rm -f "$dest.part"
		if ((attempt < attempts)); then
			sleep 5
		fi
	done
	return 1
}

# Resolve a release asset id through the GitHub API.
github_asset_id() {
	local repo="$1" tag="$2" asset="$3"
	local encoded_tag="${tag//+/%2B}"
	command -v jq >/dev/null 2>&1 || {
		github_log "jq is required to resolve GitHub release assets via the API"
		return 1
	}

	local -a curl_args=(
		--proto '=https' --tlsv1.2 -fsSL --connect-timeout 30
		--retry 3 --retry-delay 3 --retry-all-errors
	)
	if [[ -n "${GITHUB_TOKEN:-}" ]]; then
		curl_args+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
	fi

	curl "${curl_args[@]}" \
		"https://api.github.com/repos/${repo}/releases/tags/${encoded_tag}" |
		jq -r --arg name "$asset" 'first(.assets[] | select(.name == $name) | .id) // empty'
}

# Download a release asset, trying the public release URL first and the API
# asset endpoint second.
github_download_asset() {
	local repo="$1" tag="$2" asset="$3" dest="$4"
	local encoded_tag="${tag//+/%2B}"
	local base_url="https://github.com/${repo}/releases/download/${encoded_tag}"

	# Do not spend a full retry budget on a consistently failing public URL:
	# fail over to the API quickly.
	github_log "-> Downloading ${asset}: ${base_url}/${asset}"
	if GITHUB_DOWNLOAD_ATTEMPTS=2 github_download_url "$dest" "${base_url}/${asset}"; then
		return 0
	fi

	github_log "-> Public release URL unavailable; retrying via api.github.com"
	local asset_id
	asset_id=$(github_asset_id "$repo" "$tag" "$asset") || true
	if [[ -z "$asset_id" ]]; then
		github_log "Could not resolve asset '${asset}' in ${repo}@${tag}"
		return 1
	fi

	local -a api_args=(-H "Accept: application/octet-stream")
	if [[ -n "${GITHUB_TOKEN:-}" ]]; then
		api_args+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
	fi
	github_download_url "$dest" \
		"https://api.github.com/repos/${repo}/releases/assets/${asset_id}" \
		"${api_args[@]}"
}

# Download an asset and the checksums file published in the same release,
# then verify the asset's SHA-256.
github_download_verified() {
	local repo="$1" tag="$2" asset="$3" checksums="$4" dest_dir="${5:-/tmp}"
	local dest="${dest_dir}/${asset}"
	local sums="${dest_dir}/${checksums}"

	mkdir -p "$dest_dir"
	rm -f "$dest" "$dest.part" "$sums"
	github_download_asset "$repo" "$tag" "$asset" "$dest" || return 1
	if ! github_download_asset "$repo" "$tag" "$checksums" "$sums"; then
		github_log "Could not download checksums file ${checksums}"
		rm -f "$dest"
		return 1
	fi

	if ! (
		cd "$dest_dir"
		awk -v name="$asset" '$2 == name { print }' "$checksums" | sha256sum -c --status -
	); then
		github_log "Checksum verification failed for ${asset}"
		rm -f "$dest"
		return 1
	fi
	github_log "-> Verified SHA-256 checksum for ${asset}"
}
