#!/bin/sh

gen_password() {
    tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 16
}

mkdir -p secrets

if [ ! -f secrets/postgresql.env ]; then
    DB_PASSWORD=$(gen_password)
    cat > secrets/postgresql.env <<EOF
POSTGRES_PASSWORD=${DB_PASSWORD}
EOF
    echo "generated secrets/postgresql.env"
else
    echo "secrets/postgresql.env already exists, skipping"
fi

if [ ! -f secrets/server.env ]; then
    DB_PASSWORD=$(grep -s POSTGRES_PASSWORD secrets/postgresql.env | cut -d= -f2)
    AUTH_TOKEN=$(gen_password)
    cat > secrets/server.env <<EOF
DB_PASSWORD=${DB_PASSWORD}
APP_AUTH_TOKEN=${AUTH_TOKEN}
EOF
    echo "generated secrets/server.env"
else
    echo "secrets/server.env already exists, skipping"
fi