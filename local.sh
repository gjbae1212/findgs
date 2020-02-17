#!/bin/bash

set -e -o pipefail
trap '[ "$?" -eq 0 ] || echo "Error Line:<$LINENO> Error Function:<${FUNCNAME}>"' EXIT
cd `dirname $0`
CURRENT=`pwd`

function test
{
   set_env
   go test -v $(go list ./... | grep -v vendor) --count 1 -covermode=atomic --race -timeout 120s
}

function release
{
   phase=$1
   case "$phase" in
   test)
     goreleaser release --snapshot --skip-publish
     ;;
   real)
     goreleaser release --rm-dist
     ;;
   *)
     echo "unsupported phase=$phase"
     exit 1
   esac
}

function set_env
{
   source $CURRENT/local_env.sh
}

CMD=$1
shift
$CMD $*
