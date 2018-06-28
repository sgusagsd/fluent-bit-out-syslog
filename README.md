
## Fluent Bit Syslog Output Plugin

**How to Build:**

```
go get -d github.com/oratos/out_syslog/...
go build -buildmode c-shared -o out_syslog.so github.com/oratos/out_syslog/cmd
```

**How to Run:**

```
fluent-bit \
    --input dummy \
    --plugin ./out_syslog.so \
    --output syslog \
    --prop Addr=localhost:12345
```
