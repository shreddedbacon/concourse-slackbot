FROM golang AS builder
RUN go version
ENV GO111MODULE=on
COPY . /go/src/github.com/shreddedbacon/concourse-slackbot/
WORKDIR /go/src/github.com/shreddedbacon/concourse-slackbot/
# RUN go get github.com/nlopes/slack
# #v0.3.0 required, newer version has some issues
# WORKDIR /go/src/github.com/nlopes/slack
# RUN git checkout v0.3.0
WORKDIR /go/src/github.com/shreddedbacon/concourse-slackbot/
RUN set -x && \
    go get -v .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o concoursebot .

# actual container
FROM alpine:3.7
RUN apk --no-cache add ca-certificates openssl
WORKDIR /app/
# bring the actual executable from the builder
COPY --from=builder /go/src/github.com/shreddedbacon/concourse-slackbot/concoursebot .
ENTRYPOINT ["./concoursebot"]
