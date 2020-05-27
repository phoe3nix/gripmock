#!/bin/bash

if [ "$1" = "" ]; then
	echo "Version is empty"
	exit 0
fi

go build ../.

docker build -t "phoe3nix/gripmock:$1" --build-arg ssh_prv_key="$(cat ~/.ssh/id_rsa)" --build-arg ssh_pub_key="$(cat ~/.ssh/id_rsa.pub)" --build-arg ssh_hosts="$(cat ~/.ssh/known_hosts)"  --squash .
