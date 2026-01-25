#!/bin/sh
# set -x

### [global] environment ###

set -a
# env $( grep -Po '^[^;#]+' scripts/.env ) echo \$MICRO_SERVER
# source ./scripts/.env
. ./scripts/.env
set +a

### [local] environment ###

# foo=bar

### [service] command ###

go="go1.25.1" # toolchain
go="go"
# https://go.dev/doc/articles/race_detector
cmd="./bin/im-account-service"
cmd="$go run -race ."

set -x

$cmd server --config_file ./config/config.local.yml $@
