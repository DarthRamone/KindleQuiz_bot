#!/bin/sh
# migration-entrypoint.sh

set -e

host="$1"
port="$2"

echo "Connecting to $1:$2"

until PGPASSWORD=$POSTGRES_PASSWORD psql -h "$host" -U "postgres" -c '\q'; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec /go/bin/goose -dir /sql/ postgres "user=postgres dbname=vocab sslmode=disable port=$port host=$host" up