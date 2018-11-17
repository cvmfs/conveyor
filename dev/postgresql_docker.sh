#!/bin/sh

docker run -d \
    --hostname postgres00 \
    --name postgres00 \
    -e POSTGRES_PASSWORD='password' \
    -p 5432:5432 \
    postgres:11
