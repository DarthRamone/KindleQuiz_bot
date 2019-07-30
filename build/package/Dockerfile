FROM golang:1.12.7

RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o main ./cmd/kindle_quiz_bot/main.go

CMD ["./main"]