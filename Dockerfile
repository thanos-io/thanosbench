FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Thanos Authors"

COPY thanosbench /bin/thanosbench

ENTRYPOINT [ "/bin/thanosbench" ]
