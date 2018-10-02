Table of Contents
=================

   * [Table of Contents](#table-of-contents)
   * [Fluent Bit Syslog Output Plugin](#fluent-bit-syslog-output-plugin)
      * [How To Run In Local laptop](#how-to-run-in-local-laptop)
      * [How To Run Linter](#how-to-run-linter)
      * [How to Configure Fluent Bit Conf](#how-to-configure-fluent-bit-conf)
   * [Sample Config File](#sample-config-file)
      * [Syslog output plugin with kubernetes namespace filter](#syslog-output-plugin-with-kubernetes-namespace-filter)

# Fluent Bit Syslog Output Plugin

**How to Test:**

```
cd $GOPATH

# get the code
mkdir -p src/github.com/pivotal-cf
cd src/github.com/pivotal-cf
git clone git@github.com:pivotal-cf/fluent-bit-out-syslog.git

# get dependencies
cd $GOPATH/src
go get -d -t github.com/pivotal-cf/fluent-bit-out-syslog/cmd...

# run code build
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/cmd
go build -buildmode c-shared -o out_syslog.so .

# run test
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog
go test -v ./...
```

**How to Test in Docker-compose:**
```
cd $GOPATH/src/github.com/pivotal-cf/fluent-bit-out-syslog/
./tests/test.sh
```

## How To Run In Local laptop

```
fluent-bit \
    --input dummy \
    --plugin ./out_syslog.so \
    --output syslog \
    --prop ClusterSinks=[{"addr": "localhost:12345"}]
```

## How To Run Linter
```
./tests/run-linter.sh
```

## How to Configure Fluent Bit Conf
Add the following output section to the fluent bit configuration file. Note
that the `tls` configuration is optional and is required only if connecting to
an endpoint that supports TLS.
`Sinks` are also referred to as namespace sinks and allow to forward logs from
a particular namespace onto a syslog destination. Whereas, `ClusterSinks`
forward all logs from all namespaces to the specified syslog destination.

# Sample Config File
## Syslog output plugin with kubernetes namespace filter

```
[INPUT]
    Name              tail
    Tag               kube.*
    Path              /var/log/containers/*.log
    Parser            docker
    DB                /var/log/flb_kube.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10

[FILTER]
    Name                kubernetes
    Match               kube.*
    Kube_URL            https://kubernetes.default.svc.cluster.local:443
    Merge_Log           On
    K8S-Logging.Parser  On

[OUTPUT]
    Name syslog
    Match *
	Sinks [{"addr":"logs.papertrailapp.com:18271", "namespace": "myns", "tls":{"insecure_skip_verify":"true"}}]
	ClusterSinks [{"addr":"logs.papertrailapp.com:18271"}]
```
