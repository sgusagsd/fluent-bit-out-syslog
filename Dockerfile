FROM oratos/golang-base:1.11 as gobuilder

WORKDIR /root

ENV GOOS=linux \
    GOARCH=amd64

COPY /go.mod /go.sum /root/

RUN go version && \
    go mod download

COPY / /root/

RUN go build \
    -a \
    -installsuffix fluent \
    -buildmode c-shared \
    -o /out_syslog.so \
    -mod=readonly \
    cmd/main.go

FROM ubuntu:xenial as builder

# Fluent Bit version
ENV FLB_MAJOR 1
ENV FLB_MINOR 0
ENV FLB_PATCH 4
ENV FLB_VERSION 1.0.4

ENV DEBIAN_FRONTEND noninteractive

ENV FLB_TARBALL http://github.com/fluent/fluent-bit/archive/v$FLB_VERSION.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/

RUN apt-get update -qq \
    && apt-get install -qq \
       build-essential \
       cmake \
       make \
       wget \
       unzip \
       libsystemd-dev \
       libssl-dev \
       libasl-dev \
       libsasl2-dev \
    && wget -O "/tmp/fluent-bit-${FLB_VERSION}.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_VERSION.zip" \
    && cd "fluent-bit-$FLB_VERSION"/build/ \
    && cmake --quiet \
          -DFLB_DEBUG=On \
          -DFLB_TRACE=Off \
          -DFLB_JEMALLOC=On \
          -DFLB_BUFFERING=On \
          -DFLB_TLS=On \
          -DFLB_SHARED_LIB=Off \
          -DFLB_EXAMPLES=Off \
          -DFLB_HTTP_SERVER=On \
          -DFLB_OUT_KAFKA=On .. \
    && make --quiet \
    && install bin/fluent-bit /fluent-bit/bin/

# Configuration files
COPY /config/fluent-bit.conf \
     /config/parsers.conf \
     /config/parsers_java.conf \
     /config/parsers_mult.conf \
     /config/parsers_openstack.conf \
     /config/parsers_cinder.conf \
     /fluent-bit/etc/

FROM ubuntu:xenial

RUN groupadd --system fluent-bit --gid 1000 && \
    useradd --no-log-init --system --gid fluent-bit fluent-bit --uid 1000

RUN apt-get update \
    && apt-get install --no-install-recommends ca-certificates libssl1.0.2 -qq libsasl2-2 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoclean

COPY --from=builder /fluent-bit /fluent-bit
COPY --from=gobuilder /out_syslog.so /fluent-bit/bin/

EXPOSE 2020
USER 1000:1000

CMD ["/fluent-bit/bin/fluent-bit", "--plugin", "/fluent-bit/bin/out_syslog.so", "--config", "/fluent-bit/etc/fluent-bit.conf"]
