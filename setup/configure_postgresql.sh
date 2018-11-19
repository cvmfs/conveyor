#!/bin/sh

echo "listen_addresses = '*'" >> /var/lib/pgsql/data/postgresql.conf
echo "host all all 0.0.0.0/0 md5" >> /var/lib/pgsql/data/pg_hba.conf
