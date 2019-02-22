#!/bin/sh
set -e -u -x
# Install git for go get

echo ">> Install git"
apk add --no-cache git

# set up directory stuff for golang
echo ">> Setup Directories"
mkdir -p /go/src/github.com/shreddedbacon/
ln -s $PWD/concoursebot-release /go/src/github.com/shreddedbacon/concourse-slackbot
go get github.com/nlopes/slack
#v0.3.0 required, newer version has some issues
cd /go/src/github.com/nlopes/slack
git checkout v0.3.0
cd  /go/src/github.com/shreddedbacon/concourse-slackbot
echo ">> Get"
go get -v .
cd -
echo ">> Build"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o built-release/concoursebot github.com/shreddedbacon/concourse-slackbot

echo ">> Create artifact"
VERSION=$(cat ${VERSION_FROM})
cp -r /go/src/github.com/shreddedbacon/concourse-slackbot/templates built-release/
cd built-release
tar czf concoursebot-linux-$VERSION.tar.gz concoursebot
