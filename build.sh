#!/bin/bash

if [ "$1" = "" ]; then
	echo "Version is empty"
	exit 0
fi

go build ../.

docker build -t "phoe3nix/gripmock:$1" .
