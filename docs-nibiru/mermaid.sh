#!/usr/bin/env bash

# Run me with: bash mermaid.sh Install: docker pull minlag/mermaid-cli

path_to_diagrams="./mermaid-diagrams"
diagram_files=(
  "adaptive-exec-4-static-conflict-graph.mmd"
  "adaptive-exec-5-tx-approaches.mmd"
  "adaptive-exec-6-versus-traditional.mmd"
  "qrc-flow-1.mmd"
  "qrc-flow-2.mmd"
  "qrc-flow-3.mmd"
  "qrc-flow-4.mmd"
  "qrc-flow-5.mmd"
  "qrc-flow-6.mmd"
  "qrc-flow-7.mmd"
  "qrc-flow-8.mmd"
  "qrc-flow-9.mmd"
  "qrc-flow-10.mmd"
  "qrc-flow-11.mmd"
  "validator-clusters-cometbft-overhead.mmd"
  "validator-clusters-clustering.mmd"
  "validator-clusters-teams.mmd"
)

set -e

# which_ok: Check if the given binary is in the $PATH.
# Returns code 0 on success and code 1 if the command fails.
which_ok() {
  if which "$1" >/dev/null 2>&1; then
    return 0
  else
    log_error "$1 is not present in \$PATH"
    return 1
  fi
}

which_ok docker

common_frontmatter=$(cat <<'EOF'
---
config:
  theme: base
  themeVariables:
    primaryColor: '#F2BDD6'
    primaryTextColor: '#000000'
    primaryBorderColor: '#F2BDD6'
    lineColor: '#F2BDD6'
    fontSize: '16px'
    secondaryTextColor: '#000000'

    mainBkg: '#F7DBE4'
    secondBkg: '#F7DBE4'
    # --------------- unused ---------------
    background: '#F7DBE4'
---
EOF
)

inject_frontmatter() {
  local input_file="$1"
  local temp_file
  temp_file=$(mktemp)

  awk -v new_yaml="$common_frontmatter" '
    BEGIN { inside = 0 }
    {
      if ($0 ~ /^---$/ && inside == 0) {
        print new_yaml
        inside = 1
        skip = 1
        next
      }
      if ($0 ~ /^---$/ && inside == 1) {
        skip = 0
        next
      }
      if (skip == 0) print
    }
  ' "$input_file" > "$temp_file"

  mv "$temp_file" "$input_file"
}


# Help command: docker run minlag/mermaid-cli --help
for diagram_file in "${diagram_files[@]}"; do 
  input_file="$path_to_diagrams/$diagram_file"
  inject_frontmatter "$input_file"

  png_fname="${diagram_file%.*}.svg"
  docker run --rm -u "$(id -u):$(id -g)" -v "$path_to_diagrams":/data minlag/mermaid-cli \
    -i "$diagram_file" \
    -o "$png_fname" \
    --cssFile="/data/index.css"
done

mv mermaid-diagrams/qrc*.svg docs/arch/advanced/img/
mv mermaid-diagrams/adaptive-exec*.svg docs/arch/execution/img/
mv mermaid-diagrams/validator-clusters*.svg docs/arch/nibiru-bft/
