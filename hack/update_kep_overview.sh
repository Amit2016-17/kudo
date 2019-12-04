#!/usr/bin/env bash

SCRIPT_HOME=$(dirname "$0")

# Files to be included in overview generation
KEP_FILES="$SCRIPT_HOME/../keps/????-*"

# Target file
KEP_OVERVIEW_FILE="$SCRIPT_HOME/../keps/README.md"

# Overview header
cat <<EOT > $KEP_OVERVIEW_FILE
# KUDO Enhancement Proposals

*This file is autogenerated*

*Please run /hack/update_kep_overview.sh after adding or updating a KEP*

| KEP | Status | Description |
| --- | ---: | --- |
EOT

for KEP in $KEP_FILES;
do
  KEP_FILE=$(basename "$KEP")

  echo "Processing $KEP_FILE"

  # Parse KEP-Header (the part between the '---')
  KEP_HEADER=$(awk '/---/{p++} p==2{print; exit} p>=1' "$KEP")

  # Extract fields from header
  KEP_NUMBER=$(echo "$KEP_HEADER" | sed -n -E 's/kep-number: ([0-9]+)/\1/p')
  KEP_NUMBER=$(echo "$KEP_NUMBER" | sed 's/^0*//')  # Strip leading zeros, or we interpret in octal
  KEP_NUMBER=$(printf %04d "$KEP_NUMBER")           # Fill up to 4 digits again

  KEP_TITLE=$(echo "$KEP_HEADER" | sed -n -E 's/title: (.*)/\1/p')
  KEP_STATUS=$(echo "$KEP_HEADER" | sed -n -E 's/status: (.*)/\1/p')
  KEP_DESC=$(echo "$KEP_HEADER" | sed -n -E 's/short-desc: (.*)/\1/p')

  # Print one line for this KEP
  echo "| [$KEP_NUMBER - $KEP_TITLE]($KEP_FILE) | $KEP_STATUS | $KEP_DESC |" >> $KEP_OVERVIEW_FILE
done