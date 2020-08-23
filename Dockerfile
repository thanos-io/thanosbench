FROM golang:1.14.2-alpine3.11 as builder
WORKDIR $GOPATH/src/github.com/thanos-io/thanosbench
# Change in the docker context invalidates the cache so to leverage docker
# layer caching, moving update and installing apk packages above COPY cmd
# More info https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#leverage-build-cache
RUN apk update && apk upgrade && apk add --no-cache alpine-sdk
# Replaced ADD with COPY as add is generally to download content form link or tar files
# while COPY supports the basic copying of local files into the container.
# https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#add-or-copy
COPY . $GOPATH/src/github.com/thanos-io/thanosbench
RUN git update-index --refresh; make build
# -----------------------------------------------------------------------------
FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Thanos Authors"
COPY --from=builder /go/src/github.com/thanos-io/thanosbench/thanosbench /bin/thanosbench
ENTRYPOINT [ "/bin/thanosbench" ]