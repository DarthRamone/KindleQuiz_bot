FROM golang:1.12.7

RUN go get -u github.com/pressly/goose/cmd/goose

COPY deployments/migration-entrypoint.sh /

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-client \
    && rm -rf /var/lib/apt/lists/*

ENV PG_HOST "postgres"
ENV PG_PORT "5432"

RUN mkdir /sql
COPY ./sql/migrations/ /sql

CMD ["sh", "-c", "/migration-entrypoint.sh $PG_HOST $PG_PORT"]