#!/bin/sh
#set -eu -o pipefail

header() {
	echo "########################"
	echo $*
	echo
}


MAINDIR=`pwd`
VERSION=$(cat ${VERSION_FROM})
export GO111MODULE=on
# set up directory stuff for golang
header "Setup"
mkdir -p /go/src/github.com/shreddedbacon/
ln -s $PWD/concoursebot-release /go/src/github.com/shreddedbacon/concourse-slackbot
header "Get deps"
# go get github.com/nlopes/slack
# #v0.3.0 required, newer version has some issues
# cd /go/src/github.com/nlopes/slack
# git checkout v0.3.0 > /dev/null 2>&1
cd  /go/src/github.com/shreddedbacon/concourse-slackbot
go get -v .
header "Build concourse-slackbot"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o built-release/concoursebot .

header "Create artifact"
cd built-release
tar czf concoursebot-linux-dev.tar.gz concoursebot
cd $MAINDIR
ls built-release
