FROM golang:1.12.7

RUN go get -u github.com/pressly/goose/cmd/goose

RUN mkdir sql

COPY deployments/wait-for-postgres.sh /

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-client \
    && rm -rf /var/lib/apt/lists/*