#!/bin/sh

set -a
. configs/server.local.env
. secrets/server.env
set +a

make cmd/server