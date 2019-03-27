FROM centos:7

RUN yum install -y https://ecsft.cern.ch/dist/cvmfs/cvmfs-release/cvmfs-release-latest.noarch.rpm && \
    yum upgrade && \
    yum install curl cvmfs cvmfs-server cvmfs-gateway && \
    systemctl enable cvmfs-gateway
RUN curl -o /usr/local/bin/mc https://dl.minio.io/client/mc/release/linux-amd64/mc && \
    chmod +x /usr/local/bin/mc

COPY s3.conf /etc/s3cvmfs.conf

EXPOSE 4929

ENTRYPOINT [ "/usr/local/bin/start_gateway.sh" ]
