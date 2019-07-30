FROM golang:1.12.7

RUN go get -u github.com/pressly/goose/cmd/goose

RUN mkdir sql

ENTRYPOINT ["/go/bin/goose"]

CMD ["-dir", "./sql", "postgres", "\"user=postgres dbname=vocab sslmode=disable port=5432 host=postgres\"", "up"]