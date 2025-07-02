#!/bin/bash

# validate-parser-output.sh
# Script to validate that parser outputs conform to the standardized JSON structure
# Supports both compact (single-line) and pretty-printed (multi-line) JSON formats
#
# Supported formats:
#   - Single JSON object (compact or pretty-printed)
#   - JSON array containing multiple objects
#   - Concatenated JSON objects (newline-separated, pretty-printed or compact)
#   - Live parser output via stdin
#
# Usage: ./validate-parser-output.sh [<parser_output_file>]
#        <command> | ./validate-parser-output.sh
#
# Examples:
#   ./validate-parser-output.sh parser-output.json
#   ./validate-parser-output.sh sample-data.json
#   cat output.json | ./validate-parser-output.sh
#   ./parser -input data.txt | ./validate-parser-output.sh

set -e

# Function to validate a single JSON object
validate_entry() {
    local entry="$1"

    # Check required top-level fields exist
    if ! echo "$entry" | jq -e '.data_type' >/dev/null 2>&1; then
        echo "ERROR: Missing required field 'data_type'"
        return 1
    fi

    if ! echo "$entry" | jq -e '.timestamp' >/dev/null 2>&1; then
        echo "ERROR: Missing required field 'timestamp'"
        return 1
    fi

    if ! echo "$entry" | jq -e '.date' >/dev/null 2>&1; then
        echo "ERROR: Missing required field 'date'"
        return 1
    fi

    if ! echo "$entry" | jq -e '.message' >/dev/null 2>&1; then
        echo "ERROR: Missing required field 'message'"
        return 1
    fi

    # Validate data_type format (should not be empty)
    local data_type=$(echo "$entry" | jq -r '.data_type')
    if [[ -z "$data_type" || "$data_type" == "null" ]]; then
        echo "ERROR: data_type cannot be empty"
        return 1
    fi

    # Validate timestamp format (should be ISO 8601)
    local timestamp=$(echo "$entry" | jq -r '.timestamp')
    if ! echo "$timestamp" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'; then
        echo "ERROR: timestamp must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ), got: $timestamp"
        return 1
    fi

    # Validate date format (should be YYYY-MM-DD)
    local date=$(echo "$entry" | jq -r '.date')
    if ! echo "$date" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'; then
        echo "ERROR: date must be in YYYY-MM-DD format, got: $date"
        return 1
    fi

    # Validate message is an object
    if ! echo "$entry" | jq -e '.message | type == "object"' >/dev/null 2>&1; then
        echo "ERROR: message field must be an object"
        return 1
    fi

    return 0
}

# Main validation logic
entry_count=0
error_count=0

# Read input (either from file or stdin)
input_source="${1:--}"

echo "Parsing JSON objects from input..."

# Read the entire input
input_content=$(cat "$input_source")

# Function to extract JSON objects using jq and handle different formats
extract_and_validate() {
    local content="$1"

    # First, try parsing as a complete JSON structure (array or single object)
    if echo "$content" | jq -e . >/dev/null 2>&1; then
        # Check if it's an array
        if echo "$content" | jq -e 'type == "array"' >/dev/null 2>&1; then
            echo "Detected JSON array format"
            while IFS= read -r entry; do
                entry_count=$((entry_count + 1))
                if ! validate_entry "$entry"; then
                    echo "  Entry #$entry_count: $entry"
                    error_count=$((error_count + 1))
                fi
            done < <(echo "$content" | jq -c '.[]')
            return
        fi

        # Check if it's a single object by counting opening braces at start of lines
        local object_starts=$(echo "$content" | grep -c '^{')
        if [[ $object_starts -eq 1 ]]; then
            echo "Detected single JSON object format"
            entry_count=1
            entry=$(echo "$content" | jq -c '.')
            if ! validate_entry "$entry"; then
                echo "  Entry #$entry_count: $entry"
                error_count=1
            fi
            return
        fi
    fi

    # Handle concatenated JSON objects - split on lines starting with '{'
    echo "Detected concatenated JSON objects (found $object_starts objects)"

    # Use awk to split the content into separate JSON objects
    local temp_dir=$(mktemp -d)
    local object_num=0

    # Split the input into separate JSON objects using awk
    echo "$content" | awk '
        /^{/ { 
            if (object_num > 0) close(output_file)
            object_num++
            output_file = "'"$temp_dir"'/object_" object_num ".json"
        }
        { if (output_file) print > output_file }
        END { if (output_file) close(output_file) }
    '

    # Validate each extracted JSON object
    for json_file in "$temp_dir"/object_*.json; do
        if [[ -f "$json_file" ]]; then
            entry_count=$((entry_count + 1))

            # Read and validate the JSON object
            local json_content=$(cat "$json_file")
            if echo "$json_content" | jq -e . >/dev/null 2>&1; then
                local compact_entry=$(echo "$json_content" | jq -c '.')
                if ! validate_entry "$compact_entry"; then
                    echo "  Entry #$entry_count: $compact_entry"
                    error_count=$((error_count + 1))
                fi
            else
                echo "WARNING: Invalid JSON in object #$entry_count:"
                cat "$json_file"
                error_count=$((error_count + 1))
            fi
        fi
    done

    # Cleanup
    rm -rf "$temp_dir"
}

# Perform extraction and validation
extract_and_validate "$input_content"

# Report results
echo "Validation complete:"
echo "  Total entries processed: $entry_count"
echo "  Errors found: $error_count"

if [[ $error_count -eq 0 ]]; then
    echo "✅ All entries conform to the standardized format!"
    exit 0
else
    echo "❌ $error_count entries have validation errors"
    exit 1
fi
