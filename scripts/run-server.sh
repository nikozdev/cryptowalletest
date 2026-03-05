#!/bin/sh

set -a
. configs/server.local.env
. secrets/server.env
set +a

docker compose up postgresql -d

make cmd/server
./build/cmd/server.exe