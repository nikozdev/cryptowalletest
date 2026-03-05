#!/bin/sh

set -a
. secrets/server.env
set +a

make cmd/client
./build/cmd/client.exe "$@"