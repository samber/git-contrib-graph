#!/bin/bash

cd /git
git log --oneline --pretty='€{"cid":"%h", "author":"%aE", "date": "%ai", "changes": "' --stat |grep -v \| | tr "\n" " " | tr '€' "\n" |  sed '/^$/d' | sed -E 's#$#\"}#g' | jq --slurp 'group_by(.author) | .[] | sort_by(.date)' | jq 'map(.date |= split(" ")[0])' | jq 'map(.changes = (.changes | sub(" +"; "") | {"files": (. | split(" ")[0]), "insertion": (. | split(" ")[3]), "deletion": (. | split(" ")[5])}))' | jq -c '.'
