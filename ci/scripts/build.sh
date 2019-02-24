#!/bin/sh
set -eu -o pipefail

header() {
	echo
	echo "########################"
	echo $*
	echo
}

MAINDIR=`pwd`
VERSION=$(cat ${VERSION_FROM})
# set up directory stuff for golang
header "Setup Directories"
mkdir -p /go/src/github.com/shreddedbacon/
ln -s $PWD/concoursebot-release /go/src/github.com/shreddedbacon/concourse-slackbot
go get github.com/nlopes/slack
#v0.3.0 required, newer version has some issues
cd /go/src/github.com/nlopes/slack
git checkout v0.3.0
cd  /go/src/github.com/shreddedbacon/concourse-slackbot
header "Get"
go get -v .
cd $MAINDIR
header ">> Build"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o built-release/concoursebot github.com/shreddedbacon/concourse-slackbot

header "Create artifact"
cd built-release
tar czf concoursebot-linux-$VERSION.tar.gz concoursebot
