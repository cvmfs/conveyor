#!/bin/sh

echo "Starting CernVM-FS repository gateway"

mc config host add minio $MINIO_URL $MINIO_ACCESS_KEY $MINIO_SECRET_KEY
mc mb minio/cvmfs

for repository in $@ ; do
    echo "  Creating repository: $repository"
    cvmfs_server mkfs -s /etc/s3cvmfs.conf -w http://$MINIO_URL/cvmfs $repository
done

cp -rv /etc/cvmfs/keys/*.{crt,pub,gw} /shared/

echo "Starting systemd services"
/usr/sbin/init