FROM alpine:3.5
MAINTAINER Eric Stroczynski <eric.stroczynski@coreos.com>

# Utils a user might need while running grafiti in a container
RUN apk add --no-cache jq

# Copy built binary into image
COPY _output/bin/grafiti /usr/local/bin/grafiti

WORKDIR /

CMD ["grafiti", "--help"]
