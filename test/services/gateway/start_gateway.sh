#!/bin/sh

echo "Waiting for MINIO backend to become online"
sleep 5

mc config host add minio http://$MINIO_URL $MINIO_ACCESS_KEY $MINIO_SECRET_KEY
mc mb minio/cvmfs
mc policy download minio/cvmfs

rm -rv /var/spool/cvmfs
ln -s /shared /var/spool/cvmfs

idx=0
for repository in $@ ; do
    echo "  Creating repository: $repository"
    cvmfs_server mkfs -o root -s /etc/s3cvmfs.conf -w http://$MINIO_URL/cvmfs $repository
    echo "plain_text key${idx} $(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)" > /etc/cvmfs/keys/${repository}.gw
    idx=$((idx + 1))
done

cp -rv /etc/cvmfs/keys/*.{crt,pub,gw} /shared/

echo "Starting systemd services"
#/usr/sbin/init