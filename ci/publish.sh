#!/bin/sh

# publishes the compiled binary to the bucket repository
set -e

# args:
files=""          # -a   path to file(s) to add
branch=""         # -b   branch name
publish_user=""   # -u   publisher's username
publish_token=""  # -t   publisher's token
repo=""           # -r   short repo name - e.g. "owner/repo"
message=""        # -m   commit message

while getopts 'a:b:u:t:r:m:' opt; do
    case "$opt" in
        a)
            for f in $OPTARG; do
                files="$files $(realpath "$f")"
            done ;;
        b)
            branch="$OPTARG" ;;
        u)
            publish_user="$OPTARG" ;;
        t)
            publish_token="$OPTARG" ;;
        r)
            repo="$OPTARG" ;;
        m)
            message="$OPTARG" ;;
        *)
            # ignore invalid args
            echo "invalid flag: $opt" ;;
    esac
done

# validate input
for var in "$files" "$branch" "$publish_user" "$publish_token" "$repo"; do
    if [ -z "$var" ]; then
        echo "some of the variables are not provided!"
        exit 1
    fi
done

# prepare temporary directory
tempdir="$(mktemp -d)"
cd "$tempdir" || exit 1

# clone
echo "cloning bucket repository"
git clone https://"$publish_user":"$publish_token"@github.com/"$repo" bucket
cd bucket || exit 1
git config user.name "Github Actions"
git config user.email "actions@github.com"

# new branch
git checkout -b "$branch" 2>/dev/null || git checkout "$branch"

# add files to ./bin/ subdir
echo "applying changes"
mkdir -p bin/
# copy files
for f in $files; do
    cp -r "$f" bin/
done
git add bin/
[ -z "$message" ] && message="added $files"
git commit -m "$message"

# try publishing 10 times
echo "trying to push to bucket repository..."
for i in 1 2 3 4 5 6 7 8 9 10 11; do
    echo "attempt $i/10"
    if (git push -u origin "$branch"); then
        echo "push succeeded after $i attempts"
        break
    fi

    git pull origin "$branch" --rebase || true

    if [ "$i" -eq 11 ]; then
        echo "push failed after 10 attempts"
        exit 1
    fi

    sleep 3
done
