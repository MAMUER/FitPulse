#!/bin/bash
# scripts/check-server-health.sh
# Server Health Check Script with Telegram Notifications
# Usage: ./check-server-health.sh [--silent]
#   --silent: Only send notifications on critical issues

set -euo pipefail

# Configuration
SILENT_MODE=false
THRESHOLD_MEM_CRITICAL=90
THRESHOLD_MEM_WARNING=75
THRESHOLD_DISK_CRITICAL=90
THRESHOLD_DISK_WARNING=80
THRESHOLD_SWAP_ENTRIES=1

# Telegram configuration (from environment or secrets)
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_CHAT_ID="${TELEGRAM_CHAT_ID:-}"

# Parse arguments
if [[ "${1:-}" == "--silent" ]]; then
	SILENT_MODE=true
fi

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Logging function
log() {
	echo -e "${2:-}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Send Telegram notification
send_telegram() {
	local message="$1"
	local parse_mode="${2:-HTML}"
	
	if [[ -z "$TELEGRAM_BOT_TOKEN" || -z "$TELEGRAM_CHAT_ID" ]]; then
		log "Telegram credentials not configured, skipping notification" "$YELLOW"
		return 0
	fi
	
	local response
	response=$(curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-H "Content-Type: application/json" \
		-d "{
			\"chat_id\": \"${TELEGRAM_CHAT_ID}\",
			\"text\": \"${message}\",
			\"parse_mode\": \"${parse_mode}\"
		}" 2>&1)
	
	if echo "$response" | grep -q '"ok":true'; then
		log "Telegram notification sent successfully" "$GREEN"
	else
		log "Failed to send Telegram notification: $response" "$RED"
	fi
}

# Check /etc/fstab for duplicate swap entries
check_fstab_swap() {
	local swap_count
	swap_count=$(grep -c "/swapfile none swap" /etc/fstab 2>/dev/null || echo "0")
	
	if [[ "$swap_count" -gt "$THRESHOLD_SWAP_ENTRIES" ]]; then
		log "CRITICAL: Found $swap_count swap entries in /etc/fstab (should be 1)" "$RED"
		echo "CRITICAL_FSTAB:$swap_count"
		return 1
	elif [[ "$swap_count" -eq 0 ]]; then
		log "WARNING: No swap entry found in /etc/fstab" "$YELLOW"
		echo "WARNING_FSTAB:0"
		return 0
	else
		log "OK: Swap entry count in /etc/fstab: $swap_count" "$GREEN"
		echo "OK_FSTAB:$swap_count"
		return 0
	fi
}

# Check memory usage
check_memory() {
	local mem_info
	mem_info=$(free | grep Mem)
	local total used available
	total=$(echo "$mem_info" | awk '{print $2}')
	used=$(echo "$mem_info" | awk '{print $3}')
	available=$(echo "$mem_info" | awk '{print $7}')
	
	local usage_percent
	usage_percent=$(awk "BEGIN {printf \"%.0f\", ($used/$total)*100}")
	
	if [[ "$usage_percent" -gt "$THRESHOLD_MEM_CRITICAL" ]]; then
		log "CRITICAL: Memory usage ${usage_percent}% (available: ${available}KB)" "$RED"
		echo "CRITICAL_MEM:${usage_percent}:${available}"
		return 1
	elif [[ "$usage_percent" -gt "$THRESHOLD_MEM_WARNING" ]]; then
		log "WARNING: Memory usage ${usage_percent}% (available: ${available}KB)" "$YELLOW"
		echo "WARNING_MEM:${usage_percent}:${available}"
		return 0
	else
		log "OK: Memory usage ${usage_percent}% (available: ${available}KB)" "$GREEN"
		echo "OK_MEM:${usage_percent}:${available}"
		return 0
	fi
}

# Check disk usage
check_disk() {
	local disk_usage
	disk_usage=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')
	local available
	available=$(df -h / | tail -1 | awk '{print $4}')
	
	if [[ "$disk_usage" -gt "$THRESHOLD_DISK_CRITICAL" ]]; then
		log "CRITICAL: Disk usage ${disk_usage}% (available: ${available})" "$RED"
		echo "CRITICAL_DISK:${disk_usage}:${available}"
		return 1
	elif [[ "$disk_usage" -gt "$THRESHOLD_DISK_WARNING" ]]; then
		log "WARNING: Disk usage ${disk_usage}% (available: ${available})" "$YELLOW"
		echo "WARNING_DISK:${disk_usage}:${available}"
		return 0
	else
		log "OK: Disk usage ${disk_usage}% (available: ${available})" "$GREEN"
		echo "OK_DISK:${disk_usage}:${available}"
		return 0
	fi
}

# Check k3s status
check_k3s() {
	if ! command -v k3s &>/dev/null; then
		log "k3s not installed, skipping k3s check" "$YELLOW"
		echo "SKIP_K3S"
		return 0
	fi
	
	if ! systemctl is-active --quiet k3s; then
		log "CRITICAL: k3s service is not running" "$RED"
		echo "CRITICAL_K3S:stopped"
		return 1
	fi
	
	if ! k3s kubectl cluster-info &>/dev/null; then
		log "CRITICAL: k3s cluster is not responding" "$RED"
		echo "CRITICAL_K3S:unresponsive"
		return 1
	fi
	
	local node_count
	node_count=$(k3s kubectl get nodes --no-headers 2>/dev/null | wc -l)
	local ready_nodes
	ready_nodes=$(k3s kubectl get nodes --no-headers 2>/dev/null | grep -c " Ready" || echo "0")
	
	if [[ "$ready_nodes" -eq 0 ]]; then
		log "CRITICAL: No k3s nodes are ready" "$RED"
		echo "CRITICAL_K3S:no_ready_nodes"
		return 1
	fi
	
	log "OK: k3s cluster healthy ($ready_nodes/$node_count nodes ready)" "$GREEN"
	echo "OK_K3S:${ready_nodes}:${node_count}"
	return 0
}

# Check systemd failed units
check_systemd() {
	local failed_count
	failed_count=$(systemctl --failed --no-legend 2>/dev/null | wc -l)
	
	if [[ "$failed_count" -gt 0 ]]; then
		log "WARNING: $failed_count systemd units failed" "$YELLOW"
		local failed_units
		failed_units=$(systemctl --failed --no-legend 2>/dev/null | awk '{print $2}' | head -5 | tr '\n' ', ')
		echo "WARNING_SYSTEMD:${failed_count}:${failed_units}"
		return 0
	else
		log "OK: No systemd units failed" "$GREEN"
		echo "OK_SYSTEMD:0"
		return 0
	fi
}

# Check swap usage
check_swap() {
	local swap_info
	swap_info=$(free | grep Swap)
	local total used
	total=$(echo "$swap_info" | awk '{print $2}')
	used=$(echo "$swap_info" | awk '{print $3}')
	
	if [[ "$total" -eq 0 ]]; then
		log "WARNING: No swap configured" "$YELLOW"
		echo "WARNING_SWAP:none"
		return 0
	fi
	
	local usage_percent
	usage_percent=$(awk "BEGIN {printf \"%.0f\", ($used/$total)*100}")
	
	if [[ "$usage_percent" -gt 50 ]]; then
		log "WARNING: Swap usage ${usage_percent}% (${used}KB/${total}KB)" "$YELLOW"
		echo "WARNING_SWAP:${usage_percent}"
		return 0
	else
		log "OK: Swap usage ${usage_percent}% (${used}KB/${total}KB)" "$GREEN"
		echo "OK_SWAP:${usage_percent}"
		return 0
	fi
}

# Main health check
main() {
	log "Starting server health check..."
	
	local issues=()
	local critical_issues=()
	local warnings=()
	local status="HEALTHY"
	
	# Run all checks
	local fstab_result mem_result disk_result k3s_result systemd_result swap_result

	fstab_result=$(check_fstab_swap) || critical_issues+=("fstab:$fstab_result")
	mem_result=$(check_memory) || {
		[[ "$mem_result" == CRITICAL* ]] && critical_issues+=("memory:$mem_result") || warnings+=("memory:$mem_result")
	}
	disk_result=$(check_disk) || {
		[[ "$disk_result" == CRITICAL* ]] && critical_issues+=("disk:$disk_result") || warnings+=("disk:$disk_result")
	}
	k3s_result=$(check_k3s) || critical_issues+=("k3s:$k3s_result")
	systemd_result=$(check_systemd) || warnings+=("systemd:$systemd_result")
	swap_result=$(check_swap) || warnings+=("swap:$swap_result")
	
	# Determine overall status
	if [[ ${#critical_issues[@]} -gt 0 ]]; then
		status="CRITICAL"
	elif [[ ${#warnings[@]} -gt 0 ]]; then
		status="WARNING"
	fi
	
	log "Health check completed. Status: $status"
	
	# Send notification if needed
	if [[ "$SILENT_MODE" == "false" || "$status" == "CRITICAL" ]]; then
		local message="<b>🖥️ Server Health Check</b>\n"
		message+="<code>$(hostname)</code>\n"
		message+="<code>$(date '+%Y-%m-%d %H:%M:%S')</code>\n\n"
		
		case "$status" in
			"CRITICAL")
				message+="🔴 <b>Status: CRITICAL</b>\n\n"
				message+="<b>Critical Issues:</b>\n"
				for issue in "${critical_issues[@]}"; do
					message+="  ❌ ${issue}\n"
				done
				;;
			"WARNING")
				message+="🟡 <b>Status: WARNING</b>\n\n"
				message+="<b>Warnings:</b>\n"
				for warning in "${warnings[@]}"; do
					message+="  ⚠️ ${warning}\n"
				done
				;;
			"HEALTHY")
				message+="🟢 <b>Status: HEALTHY</b>\n\n"
				message+="All checks passed successfully ✅\n"
				;;
		esac
		
		# Add summary
		message+="\n<b>Summary:</b>\n"
		message+="  • Fstab: ${fstab_result%%:*}\n"
		message+="  • Memory: ${mem_result%%:*}\n"
		message+="  • Disk: ${disk_result%%:*}\n"
		message+="  • k3s: ${k3s_result%%:*}\n"
		message+="  • Systemd: ${systemd_result%%:*}\n"
		message+="  • Swap: ${swap_result%%:*}\n"
		
		send_telegram "$message"
	fi
	
	# Exit with appropriate code
	if [[ "$status" == "CRITICAL" ]]; then
		exit 1
	else
		exit 0
	fi
}

main "$@"
