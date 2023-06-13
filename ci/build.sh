#!/bin/sh

# builds a go binary with the provided args
set -e

# args:
goos=""       # -o   GOOS value
goarch=""     # -a   GOARCH value
crossarch=""  # -c   cgo cross compilation target
buildtags=""  # -b   build arguments
cgo=0         # -e   cgo enabled (true or false)
output=""     # -p   output path

while getopts 'o:a:c:b:p:e:' opt; do
    case "$opt" in
        o)
            goos="$OPTARG" ;;
        a)
            goarch="$OPTARG" ;;
        c)
            crossarch="$OPTARG" ;;
        b)
            buildtags="$OPTARG" ;;
        p)
            output="$OPTARG" ;;
        e)
            [ "$OPTARG" = "true" ] && cgo=1 ;;
        *)
            # ignore invalid args
            echo "invalid flag: $opt" ;;
    esac
done

# check if cross platform is specified
if [ -n "$crossarch" ]; then
    cc="zig cc -target $crossarch"
    cxx="zig c++ -target $crossarch"
fi

# Compile
export CGO_ENABLED="$cgo"
export CC="$cc"
export CXX="$cxx"
export GOOS="$goos"
export GOARCH="$goarch"

go build -tags="$buildtags" -o "$output"
