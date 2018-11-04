#!/bin/sh

docker run -it --rm -d \
    --hostname my-rabbit \
    --name some-rabbit \
    -e RABBITMQ_ERLANG_COOKIE='secret' \
    -e RABBITMQ_NODENAME=rabbit@my-rabbit \
    -p 8080:15672 \
    rabbitmq:3-management
