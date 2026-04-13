#!/bin/bash
# compare-telemetry.sh — Compares gNMI vs CLI telemetry data from the same switch.
#
# Runs both the gnmi-collector (gNMI Get) and the cisco-parser (CLI show commands)
# for selected data types and saves the results side by side for comparison.
#
# Prerequisites:
#   - gnmi-collector binary in /opt/gnmi-collector/ or $GNMI_COLLECTOR_PATH
#   - cisco-parser binary in /opt/cisco-parser/ or $CISCO_PARSER_PATH
#   - gNMI credentials set: GNMI_USER, GNMI_PASS
#   - Run on the switch or a host with access to the switch
#
# Usage:
#   ./compare-telemetry.sh [output-file]
#   ./compare-telemetry.sh /tmp/telemetry-comparison.txt

set -uo pipefail

# Configuration
GNMI_COLLECTOR="${GNMI_COLLECTOR_PATH:-/opt/gnmi-collector/gnmi-collector}"
GNMI_CONFIG="${GNMI_CONFIG_PATH:-/opt/gnmi-collector/config.yaml}"
CISCO_PARSER="${CISCO_PARSER_PATH:-/opt/cisco-parser}"
OUTPUT_FILE="${1:-/tmp/telemetry-comparison-$(date +%Y%m%d-%H%M%S).txt}"
TMPDIR=$(mktemp -d /tmp/telemetry-compare.XXXXXX)

trap "rm -rf $TMPDIR" EXIT

# Data types to compare.
# Format: "label|gnmi_table_name|cli_command|parser_name"
# gnmi_table_name must match the "table" field in config.yaml exactly —
# the dry-run output format is: [TableName] {json...}
COMPARISONS=(
    "environment-temperature|CiscoEnvTemp_CL|show environment temperature|environment-temperature"
    "environment-power|CiscoEnvPower_CL|show environment power|environment-power"
    "interface-counters|CiscoInterfaceCounter_CL|show interface counters|interface-counters"
    "interface-status|CiscoInterfaceStatus_CL|show interface status|interface-status"
    "bgp-summary|CiscoBgpSummary_CL|show bgp all summary|bgp-all-summary"
    "system-resources|CiscoSystemResources_CL|show system resources|system-resources"
    "system-uptime|CiscoSystemUptime_CL|show system uptime|system-uptime"
    "inventory|CiscoInventory_CL|show inventory|inventory"
    "lldp-neighbor|CiscoLldpNeighbor_CL|show lldp neighbor detail|lldp-neighbor"
    "ip-arp|CiscoIpArp_CL|show ip arp|ip-arp"
    "mac-address|CiscoMacAddress_CL|show mac address-table|mac-address"
    "transceiver|CiscoTransceiver_CL|show interface transceiver detail|transceiver"
)

# Header
{
    echo "=============================================================================="
    echo "  TELEMETRY DATA COMPARISON: gNMI vs CLI"
    echo "  Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    echo "  Host: $(hostname 2>/dev/null || echo 'unknown')"
    echo "  gNMI Collector: $GNMI_COLLECTOR"
    echo "  gNMI Config: $GNMI_CONFIG"
    echo "  CLI Parser: $CISCO_PARSER"
    echo "=============================================================================="
    echo ""
} > "$OUTPUT_FILE"

# Check prerequisites
check_binary() {
    if [ ! -x "$1" ]; then
        echo "WARNING: $1 not found or not executable" | tee -a "$OUTPUT_FILE"
        return 1
    fi
    return 0
}

gnmi_available=true
cli_available=true

check_binary "$GNMI_COLLECTOR" || gnmi_available=false
check_binary "$CISCO_PARSER" || cli_available=false

if [ "$gnmi_available" = false ] && [ "$cli_available" = false ]; then
    echo "ERROR: Neither gnmi-collector nor cisco-parser found. Exiting." | tee -a "$OUTPUT_FILE"
    exit 1
fi

echo "Output will be saved to: $OUTPUT_FILE"
echo ""

# Collect gNMI data (all paths at once via dry-run --once)
if [ "$gnmi_available" = true ]; then
    echo "Collecting gNMI data..."
    echo "[DEBUG] Running: $GNMI_COLLECTOR --config $GNMI_CONFIG --dry-run --once"
    if "$GNMI_COLLECTOR" --config "$GNMI_CONFIG" --dry-run --once \
        > "$TMPDIR/gnmi-all.txt" 2> "$TMPDIR/gnmi-errors.txt"; then
        echo "  gNMI collection complete."
    else
        echo "  WARNING: gNMI collection exited with error (code $?)."
    fi

    # Debug: show what we captured
    gnmi_lines=$(wc -l < "$TMPDIR/gnmi-all.txt" 2>/dev/null || echo 0)
    gnmi_err_lines=$(wc -l < "$TMPDIR/gnmi-errors.txt" 2>/dev/null || echo 0)
    echo "  [DEBUG] gNMI stdout: $gnmi_lines lines"
    echo "  [DEBUG] gNMI stderr: $gnmi_err_lines lines"

    # Show the table names found in the output
    tables_found=$(grep -oP '^\[\K[^\]]+' "$TMPDIR/gnmi-all.txt" 2>/dev/null | sort -u || true)
    if [ -n "$tables_found" ]; then
        echo "  [DEBUG] Tables found in gNMI output:"
        echo "$tables_found" | sed 's/^/    - /'
    else
        echo "  [DEBUG] No [TableName] markers found in gNMI output."
        echo "  [DEBUG] First 10 lines of gNMI stdout:"
        head -10 "$TMPDIR/gnmi-all.txt" 2>/dev/null | sed 's/^/    > /'
        echo "  [DEBUG] First 10 lines of gNMI stderr:"
        head -10 "$TMPDIR/gnmi-errors.txt" 2>/dev/null | sed 's/^/    > /'
    fi
else
    echo "Skipping gNMI collection (binary not found)."
fi

echo ""

# extract_gnmi_entries: Extract all entries for a given table from the dry-run output.
# The dry-run format is multi-line:
#   [TableName] {
#     "data_type": "...",
#     ...
#   }
# We use awk to extract blocks starting with [TableName] and ending at the next [
extract_gnmi_entries() {
    local table_name="$1"
    local input_file="$2"

    awk -v table="$table_name" '
        # Match line starting with [TableName]
        $0 ~ "^\\[" table "\\] " {
            # Print everything after "[TableName] "
            sub("^\\[" table "\\] ", "")
            printing = 1
            print
            next
        }
        # Stop at next table marker
        /^\[/ {
            printing = 0
            next
        }
        # Print continuation lines (indented JSON)
        printing {
            print
        }
    ' "$input_file"
}

# Process each data type
for entry in "${COMPARISONS[@]}"; do
    IFS='|' read -r label gnmi_table cli_command parser_name <<< "$entry"

    {
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  $label"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        # --- gNMI Data ---
        echo ""
        echo "┌─── gNMI DATA (table: $gnmi_table) ───"
        echo "│"
        if [ "$gnmi_available" = true ] && [ -f "$TMPDIR/gnmi-all.txt" ]; then
            gnmi_data=$(extract_gnmi_entries "$gnmi_table" "$TMPDIR/gnmi-all.txt")
            if [ -n "$gnmi_data" ]; then
                echo "$gnmi_data" | sed 's/^/│  /'
                total_entries=$(echo "$gnmi_data" | grep -c '"data_type"' || true)
                echo "│"
                echo "│  [entries: $total_entries]"
            else
                echo "│  (no data returned for table: $gnmi_table)"
                echo "│  [DEBUG] grep test:"
                grep -c "$gnmi_table" "$TMPDIR/gnmi-all.txt" 2>/dev/null | sed 's/^/│    matches: /' || echo "│    matches: 0"
            fi
        else
            echo "│  (gNMI collector not available)"
        fi
        echo "│"
        echo "└───"

        # --- CLI Data ---
        echo ""
        echo "┌─── CLI DATA (command: $cli_command, parser: $parser_name) ───"
        echo "│"
        if [ "$cli_available" = true ]; then
            cli_raw="$TMPDIR/cli-raw-${label}.txt"
            cli_parsed="$TMPDIR/cli-parsed-${label}.json"

            echo "│  [DEBUG] Running: vsh -c \"$cli_command\""
            if vsh -c "$cli_command" > "$cli_raw" 2>/dev/null; then
                raw_lines=$(wc -l < "$cli_raw")
                echo "│  [DEBUG] vsh returned $raw_lines lines"

                echo "│  [DEBUG] Running: $CISCO_PARSER -p $parser_name -i $cli_raw -o $cli_parsed"
                if "$CISCO_PARSER" -p "$parser_name" -i "$cli_raw" -o "$cli_parsed" 2>/dev/null; then
                    if [ -s "$cli_parsed" ]; then
                        cat "$cli_parsed" | sed 's/^/│  /'
                        total_lines=$(wc -l < "$cli_parsed")
                        echo "│"
                        echo "│  [lines: $total_lines]"
                    else
                        echo "│  (parser produced empty output)"
                    fi
                else
                    echo "│  (parser failed for $parser_name)"
                fi
            else
                echo "│  (vsh command failed: $cli_command)"
            fi
        else
            echo "│  (cisco-parser not available)"
        fi
        echo "│"
        echo "└───"

    } >> "$OUTPUT_FILE"

    # Progress indicator
    echo "  Processed: $label"
done

# Append gNMI logs
if [ -f "$TMPDIR/gnmi-errors.txt" ] && [ -s "$TMPDIR/gnmi-errors.txt" ]; then
    {
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  gNMI COLLECTOR LOGS (stderr)"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        cat "$TMPDIR/gnmi-errors.txt"
    } >> "$OUTPUT_FILE"
fi

{
    echo ""
    echo "=============================================================================="
    echo "  END OF COMPARISON"
    echo "  File: $OUTPUT_FILE"
    echo "=============================================================================="
} >> "$OUTPUT_FILE"

echo ""
echo "Comparison complete. Results saved to: $OUTPUT_FILE"
echo "Transfer this file for analysis."
