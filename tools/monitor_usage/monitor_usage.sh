#!/bin/bash
# Monitor CPU% and Memory% and report the highest spike.
# 
# This script uses native Linux commands:
#   - /proc/stat: For CPU usage calculation
#   - free -m: For memory usage in MB
#   - Logs timestamped data to ./usage_YYYYMMDD_HHMMSS.log
#
# Usage: ./monitor_usage.sh [DURATION_SEC] [INTERVAL_SEC]
#   DURATION_SEC: Total monitoring duration in seconds (default: 3600 = 1 hour)
#   INTERVAL_SEC: Sampling interval in seconds (default: 60)
#
# Examples:
#   bash monitor_usage.sh 7200 30  # Monitor for 2 hours, sample every 30s
#   nohup bash monitor_usage.sh 1800 30 &  # Run in background for 30 mins, 30s interval
#
# Check if running (will show PID and command if found, nothing if not running):
#   pgrep -af monitor_usage
#
# Stop the script (finds PID and kills it):
#   kill $(pgrep -f monitor_usage.sh)
#
# Monitor the log in real-time:
#   tail -f usage_*.log

# Parse arguments with defaults
DURATION_SEC=${1:-3600}   # default: 1 hour
INTERVAL_SEC=${2:-60}     # default: 60 seconds

# Validate inputs
if ! [[ "$DURATION_SEC" =~ ^[0-9]+$ ]] || [ "$DURATION_SEC" -lt 1 ]; then
  echo "Error: DURATION_SEC must be a positive integer" >&2
  exit 1
fi

if ! [[ "$INTERVAL_SEC" =~ ^[0-9]+$ ]] || [ "$INTERVAL_SEC" -lt 1 ]; then
  echo "Error: INTERVAL_SEC must be a positive integer" >&2
  exit 1
fi

if [ "$INTERVAL_SEC" -gt "$DURATION_SEC" ]; then
  echo "Error: INTERVAL_SEC cannot be greater than DURATION_SEC" >&2
  exit 1
fi

ITERATIONS=$((DURATION_SEC / INTERVAL_SEC))
LOG_FILE="./usage_$(date '+%Y%m%d_%H%M%S').log"

max_cpu_pct=0
max_cpu_ts=""
max_cpu_process=""
max_mem_pct=0
max_mem_ts=""
max_mem_detail=""
max_mem_process=""

echo "=== Monitoring start: $(date '+%F %T') ===" | tee -a "$LOG_FILE"
echo "Interval: ${INTERVAL_SEC}s, Iterations: ${ITERATIONS}" | tee -a "$LOG_FILE"
echo "Timestamp, CPU_Used(%), Mem_Used(%), Mem_Detail(used/total MB), Top_Process(CPU%)" | tee -a "$LOG_FILE"

for i in $(seq 1 "$ITERATIONS"); do
  ts="$(date '+%F %T')"

  # ---- CPU used % using /proc/stat (more reliable than top)
  # Take two samples with a small delay to calculate usage
  read -r _ user1 nice1 system1 idle1 iowait1 irq1 softirq1 steal1 _ _ < /proc/stat
  sleep 0.2
  read -r _ user2 nice2 system2 idle2 iowait2 irq2 softirq2 steal2 _ _ < /proc/stat
  
  # Calculate deltas
  user_delta=$((user2 - user1))
  nice_delta=$((nice2 - nice1))
  system_delta=$((system2 - system1))
  idle_delta=$((idle2 - idle1))
  iowait_delta=$((iowait2 - iowait1))
  irq_delta=$((irq2 - irq1))
  softirq_delta=$((softirq2 - softirq1))
  steal_delta=$((steal2 - steal1))
  
  total_delta=$((user_delta + nice_delta + system_delta + idle_delta + iowait_delta + irq_delta + softirq_delta + steal_delta))
  
  # Calculate CPU usage percentage
  if [ "$total_delta" -gt 0 ]; then
    idle_pct=$(awk -v idle="$idle_delta" -v total="$total_delta" 'BEGIN { printf "%.2f", (idle * 100.0) / total }')
    cpu_used_pct=$(awk -v idle_pct="$idle_pct" 'BEGIN { printf "%.2f", 100.0 - idle_pct }')
  else
    cpu_used_pct="0.00"
  fi

  # ---- Memory used % from 'free -m'
  mem_line=$(free -m | awk '/^Mem:/ {print $2, $3}')
  total_mb=$(echo "$mem_line" | awk '{print $1}')
  used_mb=$(echo "$mem_line" | awk '{print $2}')
  
  if [ -n "$total_mb" ] && [ "$total_mb" -gt 0 ]; then
    mem_used_pct=$(awk -v u="$used_mb" -v t="$total_mb" 'BEGIN { printf "%.2f", (u * 100.0) / t }')
  else
    mem_used_pct="0.00"
  fi

  # ---- Get top CPU consuming process
  top_process=$(ps aux --sort=-%cpu | awk 'NR==2 {cmd=""; for(i=11;i<=NF;++i){cmd=cmd (i==11?"":" ") $i} printf "%s(%.1f%%)", cmd, $3}')
  if [ -z "$top_process" ]; then
    top_process="N/A"
  fi

  # ---- Log the sample
  echo "$ts, $cpu_used_pct, $mem_used_pct, ${used_mb}MB/${total_mb}MB, $top_process" | tee -a "$LOG_FILE"

  # ---- Track max CPU (optimized comparison)
  if [ "$(awk -v curr="$cpu_used_pct" -v max="$max_cpu_pct" 'BEGIN { print (curr > max) }')" = "1" ]; then
    max_cpu_pct="$cpu_used_pct"
    max_cpu_ts="$ts"
    max_cpu_process="$top_process"
  fi

  # ---- Track max Mem (optimized comparison)
  if [ "$(awk -v curr="$mem_used_pct" -v max="$max_mem_pct" 'BEGIN { print (curr > max) }')" = "1" ]; then
    max_mem_pct="$mem_used_pct"
    max_mem_ts="$ts"
    max_mem_detail="${used_mb}MB/${total_mb}MB"
    max_mem_process="$top_process"
  fi

  # Sleep until next sample unless it's the last iteration
  if [ "$i" -lt "$ITERATIONS" ]; then
    sleep "$INTERVAL_SEC"
  fi
done

echo "=== Monitoring end: $(date '+%F %T') ===" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "----- Summary (Highest Observed) -----" | tee -a "$LOG_FILE"
echo "Highest CPU Used : ${max_cpu_pct}% at ${max_cpu_ts}" | tee -a "$LOG_FILE"
echo "  Top Process   : ${max_cpu_process}" | tee -a "$LOG_FILE"
echo "Highest Mem Used : ${max_mem_pct}% at ${max_mem_ts} (${max_mem_detail})" | tee -a "$LOG_FILE"
echo "  Top Process   : ${max_mem_process}" | tee -a "$LOG_FILE"