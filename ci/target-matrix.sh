#!/bin/sh
# assembles github actions matrix from targets list

default_buildplatform="ubuntu-latest"

# handle primary platforms flag (used in pull_request CI/CD)
if [ "$1" = "--primary" ]; then
    primary_filter='[.[] | select(.primary == true)]'
else
    primary_filter='.'
fi
# strip comments
targets="$(sed '/^\s*\/\//d;s/\/\/.*//' "$(dirname "$0")/targets.json")"

# filter for primary platforms if requested
targets="$(echo "$targets" | jq "$primary_filter")"

# assign a default buildplatform
targets="$(echo "$targets" | jq 'map(
    . + if has("buildplatform") then
    {buildplatform}
    else
      {buildplatform: "'"$default_buildplatform"'"}
    end
)')"

# echo the matrix (remove newlines)
echo 'matrix={"include":'"$targets"'}' | tr -d '\n'
