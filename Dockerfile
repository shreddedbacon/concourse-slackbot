FROM golang

WORKDIR /go/src/concoursebot

COPY *.go /go/src/concoursebot/

RUN go get github.com/nlopes/slack
RUN cd /go/src/github.com/nlopes/slack && git checkout v0.3.0 && cd /go/src/concoursebot
RUN go get -v .
RUN go build -o concoursebot bot.go concourse.go structs.go

ENTRYPOINT cp /go/src/concoursebot/concoursebot /project/builds/concoursebot

#CMD ["./concoursebot"]
