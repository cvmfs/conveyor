#!/bin/sh

admin_user=$1
admin_pass=$2
worker_user=$3
worker_pass=$4
vhost_name="/cvmfs"

# Enable management plugin
rabbitmq-plugins enable rabbitmq_management

# Set VM HWM
rabbitmqctl set_vm_memory_high_watermark 0.9

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

# Add and configure the worker user
if [ x"$(rabbitmqctl list_users | grep '^${worker_user}' | wc -l)" = x"0" ]; then
    rabbitmqctl add_user ${worker_user} ${worker_pass}
    rabbitmqctl set_permissions -p $vhost_name ${worker_user} "^(amq\.gen.*|jobs.*)$" "^(amq\.gen.*|jobs.*)$" ".*"
fi

