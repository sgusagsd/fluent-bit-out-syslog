ARG BASE_IMAGE=ubuntu:bionic
FROM $BASE_IMAGE

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
       build-essential \
       cmake \
       make \
       wget \
       unzip \
       libsystemd-dev \
       libssl-dev \
       libsasl2-dev \
       flex \
       bison \
       gcc \
       git \
    && apt-get clean

# Install Go
ARG GOLANG_SOURCE=dl.google.com/go
RUN wget https://$GOLANG_SOURCE/go1.12.7.linux-amd64.tar.gz -O go.tar.gz && \
    tar -xf go.tar.gz && \
    mv go /usr/local && \
    rm go.tar.gz
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

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
ENV FLB_SHA b3adad27582ed7db0338b699391ecc6bd3779c1f
ENV FLB_TARBALL https://github.com/pivotal/fluent-bit/archive/$FLB_SHA.zip

RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/src/ \
    && wget -O "/tmp/fluent-bit-$FLB_SHA.zip" ${FLB_TARBALL} \
    && cd /tmp && unzip "fluent-bit-$FLB_SHA.zip" \
    && cd "fluent-bit-$FLB_SHA"/build/ \
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
    && install bin/fluent-bit /fluent-bit/bin/ \
    && rm -rf /tmp/fluent-bit-*

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
