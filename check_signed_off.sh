#!/bin/bash

[ $# -eq 0 ] && {
    echo "usage: $(basename $0) <git-range>"
    exit 1
}

range=$1
echo "Checking range: $range"

notsigned=
for i in $( (git rev-list --no-merges "$range") )
do
    COMMIT_MSG="$(git show -s --format=%B "$i")"
    COMMIT_USER_EMAIL="$(git show -s --format="%an <%ae>" "$i")"
    echo "$COMMIT_MSG" | grep -F "Signed-off-by: ${COMMIT_USER_EMAIL}" >/dev/null

    if [ $? -ne 0 ]; then
        echo >&2 "commit ${i} is not signed off"
        notsigned="$notsigned $i"
    fi

done

[ -z "$notsigned" ] || {
    exit 1
}
