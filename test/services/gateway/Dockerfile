# FROM centos:7

# RUN yum install -y epel-release https://ecsft.cern.ch/dist/cvmfs/cvmfs-release/cvmfs-release-latest.noarch.rpm && \
#     yum upgrade -y && \
#     yum install -y curl jq cvmfs cvmfs-server && \
#     yum clean all
# RUN yum install -y cvmfs-gateway && \
#     yum clean all && \
#     systemctl enable cvmfs-gateway
# RUN curl -o /usr/local/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/mc && \
#     chmod +x /usr/local/bin/mc

# COPY s3.conf /etc/s3cvmfs.conf
# COPY start_gateway.sh /usr/local/bin/start_gateway.sh

# EXPOSE 4929

# ENTRYPOINT [ "/usr/local/bin/start_gateway.sh" ]

FROM ubuntu:bionic

RUN apt-get update && apt-get install -y systemd lsb-release curl jq && \
    curl -O https://ecsft.cern.ch/dist/cvmfs/cvmfs-release/cvmfs-release-latest_all.deb && \
    dpkg -i cvmfs-release-latest_all.deb && rm -f cvmfs-release-latest_all.deb && apt-get update
RUN apt-get install -y cvmfs cvmfs-server
RUN curl -O http://ecsft.cern.ch/dist/cvmfs/cvmfs-gateway-1.0.0/cvmfs-gateway_1.0.0~1+ubuntu18.04_amd64.deb && \
    dpkg --unpack cvmfs-gateway_1.0.0~1+ubuntu18.04_amd64.deb && rm -f cvmfs-gateway_1.0.0~1+ubuntu18.04_amd64.deb \
    systemctl enable cvmfs-gateway
RUN curl -o /usr/local/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/mc && \
    chmod +x /usr/local/bin/mc

RUN mkdir -p /etc/cvmfs/gateway && \
    cp /usr/libexec/cvmfs-gateway/etc/repo.json /etc/cvmfs/gateway/ && \
    cp /usr/libexec/cvmfs-gateway/etc/user.json /etc/cvmfs/gateway/ && \
    mkdir -p /var/lib/cvmfs-gateway && \
    RUNNER_LOG_DIR=/var/log/cvmfs-gateway-runner /usr/libexec/cvmfs-gateway/bin/cvmfs_gateway escript scripts/setup_mnesia.escript /var/lib/cvmfs-gateway

COPY s3.conf /etc/s3cvmfs.conf
COPY start_gateway.sh /usr/local/bin/start_gateway.sh

EXPOSE 4929

# ENTRYPOINT [ "/usr/local/bin/start_gateway.sh" ]
