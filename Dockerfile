FROM ubuntu:xenial
# Install Go
ADD https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz go.tar.gz
RUN tar -xf go.tar.gz && mv go /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

RUN apt-get update \
    && apt-get install -y \
       build-essential \
       cmake \
       make \
       wget \
       unzip \
       libsystemd-dev \
       libssl-dev \
       libasl-dev \
       libsasl2-dev \
       flex \
       bison \
       gcc \
       git

ENV GOOS=linux \
    GOARCH=amd64

COPY / /syslog-plugin/

RUN cd /syslog-plugin && go build \
    -a \
    -installsuffix fluent \
    -buildmode c-shared \
    -o /syslog-plugin/out_syslog.so \
    -mod=readonly \
    -mod=vendor \
    cmd/main.go

# Fluent Bit version
ENV FLB_MAJOR 1
ENV FLB_MINOR 1
ENV FLB_PATCH 3
ENV FLB_VERSION 1.1.3

ENV DEBIAN_FRONTEND noninteractive

ENV FLB_TARBALL https://github.com/fluent/fluent-bit/archive/v$FLB_VERSION.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/

RUN wget -O "/tmp/fluent-bit-${FLB_VERSION}.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_VERSION.zip" \
    && cd "fluent-bit-$FLB_VERSION"/build/ \
    && cmake -DFLB_DEBUG=On \
          -DFLB_TRACE=Off \
          -DFLB_JEMALLOC=On \
          -DFLB_BUFFERING=On \
          -DFLB_TLS=On \
          -DFLB_SHARED_LIB=Off \
          -DFLB_EXAMPLES=Off \
          -DFLB_HTTP_SERVER=On \
          -DFLB_OUT_KAFKA=On .. \
    && make \
    && install bin/fluent-bit /fluent-bit/bin/

# Configuration files
COPY /config/fluent-bit.conf \
     /config/parsers.conf \
     /config/parsers_java.conf \
     /config/parsers_mult.conf \
     /config/parsers_openstack.conf \
     /config/parsers_cinder.conf \
     /fluent-bit/etc/

EXPOSE 2020

CMD ["/fluent-bit/bin/fluent-bit", "--plugin", "/syslog-plugin/out_syslog.so", "--config", "/fluent-bit/etc/fluent-bit.conf"]
