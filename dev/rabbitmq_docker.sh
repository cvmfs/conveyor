#!/bin/sh

docker run -d \
    --hostname rabbit00 \
    --name rabbit00 \
    -e RABBITMQ_ERLANG_COOKIE='secret' \
    -e RABBITMQ_NODENAME=rabbit@rabbit00 \
    -p 15672:15672 \
    -p 5672:5672 \
    rabbitmq:3-management
