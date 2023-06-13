#!/bin/sh
# assembles github actions matrix from targets list

default_buildplatform="ubuntu-latest"

# strip comments
targets="$(sed '/^\s*\/\//d;s/\/\/.*//' "$(dirname "$0")/targets.json")"

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
