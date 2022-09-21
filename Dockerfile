FROM golang:latest

ADD . /go/src/github.com/aarongalang/cosi-web-test

RUN go install github.com/aarongalang/cosi-web-test@latest

EXPOSE 8080

ENTRYPOINT ["/cosi-web-test"]