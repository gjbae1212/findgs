#!/bin/bash

set -e -o pipefail
trap '[ "$?" -eq 0 ] || echo "Error Line:<$LINENO> Error Function:<${FUNCNAME}>"' EXIT
cd `dirname $0`
CURRENT=`pwd`

function test
{
   set_env
   go test -v $(go list ./... | grep -v vendor) --count 1 -covermode=atomic -timeout 120s
}

function release
{
  sudo rm -rf $CURRENT/dist
  sudo rm -rf $CURRENT/gopath
  export GOPATH=$CURRENT/gopath

  tag=$1
  if [ -z "$tag" ]
  then
     echo "not found tag name"
     exit 1
  fi

  git tag -a $tag -m "Add $tag"
  git push origin $tag

  goreleaser release --rm-dist
}

function release_test
{
  sudo rm -rf $CURRENT/dist
  sudo rm -rf $CURRENT/gopath
  export GOPATH=$CURRENT/gopath
  goreleaser release --snapshot --rm-dist
}


function set_env
{
   source $CURRENT/local_env.sh
}

CMD=$1
shift
$CMD $*
