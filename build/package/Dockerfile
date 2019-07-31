FROM golang:1.12.7

RUN mkdir /app
WORKDIR /app

ADD go.mod .
ADD go.sum .
RUN go mod download
ADD . .

RUN go build -o main ./cmd/kindle_quiz_bot/main.go