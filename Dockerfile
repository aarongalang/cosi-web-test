FROM golang:latest

ADD . /go/src/github.com/aarongalang/cosi-web-test

RUN go install github.com/aarongalang/cosi-web-test@latest

ENTRYPOINT /go/bin/golang-docker

EXPOSE 2379