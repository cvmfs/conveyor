#!/bin/sh

admin_user=$1
admin_pass=$2
producer_user=$3
producer_pass=$4
consumer_user=$5
consumer_pass=$6

vhost_name="/cvmfs"

# Enable management plugin
rabbitmq-plugins enable rabbitmq_management

# Delete default guest user
if [ x"$(rabbitmqctl list_users | grep 'guest' | wc -l)" != x"0" ]; then
    rabbitmqctl delete_user guest
fi

# Add the CVMFS vhost if needed
if [ x"$(rabbitmqctl list_vhosts | grep $vhost_name | wc -l)" = x"0" ]; then
    rabbitmqctl add_vhost $vhost_name
fi

# Add and configure the administrator user
if [ x"$(rabbitmqctl list_users | grep '^${admin_user}' | wc -l)" = x"0" ]; then
    rabbitmqctl add_user ${admin_user} ${admin_pass}
    rabbitmqctl set_permissions -p $vhost_name ${admin_user} ".*" ".*" ".*"
    rabbitmqctl set_user_tags ${admin_user} administrator
fi

# Add and configure the publisher user
if [ x"$(rabbitmqctl list_users | grep '^${producer_user}' | wc -l)" = x"0" ]; then
    rabbitmqctl add_user ${producer_user} ${producer_pass}
    rabbitmqctl set_permissions -p $vhost_name ${producer_user} "^(amq\.gen.*)$" "^(amq\.gen.*)$" ".*"
fi

# Add and configure the subscriber user
if [ x"$(rabbitmqctl list_users | grep '^${consumer_user}' | wc -l)" = x"0" ]; then
    rabbitmqctl add_user ${consumer_user} ${consumer_pass}
    rabbitmqctl set_permissions -p $vhost_name ${consumer_user} "^(amq\.gen.*)$" "^(amq\.gen.*)$" ".*"
fi

