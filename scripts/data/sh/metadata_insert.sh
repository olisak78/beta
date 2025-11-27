#!/bin/bash
set -euo pipefail

# Helper script to insert attributes to metadata by reading values from text file and inserting them to a copy of landscapes.yaml (can be changed to other)

INPUT_FILE="input.txt"
YAML_FILE="../landscapes.yaml" # YAML_FILE="../components.yaml"
OUTPUT_FILE="../landscapes.updated.yaml" # OUTPUT_FILE="../components.updated.yaml"
PREFIX_VALUE="operator.operationsconsole"
#PREFIX_VALUE="operation-console.operationsconsole"
#PREFIX_VALUE="operations-console.operationsconsole"

[[ -f "$INPUT_FILE" ]] || { echo "Missing $INPUT_FILE"; exit 1; }
[[ -f "$YAML_FILE"  ]] || { echo "Missing $YAML_FILE";  exit 1; }

# Start from a fresh copy
cp -f "$YAML_FILE" "$OUTPUT_FILE"

while IFS= read -r line; do
  # take first token before first space
  first_word="${line%% *}"                     # first token
  rest="${line#* }"                            # everything after the first space
  second_word="${rest%% *}"                    # second token (may be empty)
  [[ -z "$first_word" || "${first_word:0:1}" == "#" ]] && continue

  echo "Updating: $first_word to $second_word"

  # 1) Ensure metadata exists on the matched item(s)
  # .components
  yq eval -i \
    "(.landscapes[] | select(.name == \"$first_word\")).metadata |= (. // {})" \
    "$OUTPUT_FILE"

  # 2) Set metadata.oc-prefix on the matched item(s)
  yq eval -i "(.landscapes[] | select(.name == \"$first_word\")).metadata.\"debug\" = \"$second_word\"" "$OUTPUT_FILE"
  # delete: yq eval -i "del(.landscapes[] | select(.name == \"$first_word\").metadata.debug)" "$OUTPUT_FILE"

done < "$INPUT_FILE"

echo "Done. Updated file: $OUTPUT_FILE"
