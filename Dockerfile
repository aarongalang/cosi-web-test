FROM golang:latest

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /cosi-web-test

EXPOSE 8080

CMD ["/cosi-web-test"]