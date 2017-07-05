FROM golang:1.8.3-alpine
MAINTAINER Eric Stroczynski <eric.stroczynski@coreos.com>

ENV GRAFITI_ABS_PATH ${GOPATH}/src/github.com/coreos/grafiti

COPY . ${GRAFITI_ABS_PATH}/

# Install grafiti, build utils, and jq
RUN set -eux \
    && apk add --no-cache jq make glide bash \
    && apk add --no-cache --virtual .build-deps git \
    && cd ${GRAFITI_ABS_PATH} \
    && make install \
    && apk del .build-deps
