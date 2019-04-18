package main

import (
	"C"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/web"
)

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	return output.FLBPluginRegister(
		def,
		"syslog",
		"syslog output plugin that follows RFC 5424",
	)
}

var multiStateProvider web.MultiStateProvider

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	addr := output.FLBPluginConfigKey(plugin, "addr")
	name := output.FLBPluginConfigKey(plugin, "name")
	namespace := output.FLBPluginConfigKey(plugin, "namespace")
	cluster := output.FLBPluginConfigKey(plugin, "cluster")
	tls := output.FLBPluginConfigKey(plugin, "tls")
	statsAddr := output.FLBPluginConfigKey(plugin, "statsaddr")

	if addr == "" {
		log.Println("[out_syslog] ERROR: Addr is required")
		return output.FLB_ERROR
	}
	if name == "" {
		log.Println("[out_syslog] ERROR: Name is required")
		return output.FLB_ERROR
	}
	if statsAddr == "" {
		log.Println("[out_syslog] ERROR: StatsAddr is required")
		return output.FLB_ERROR
	}

	var once sync.Once
	once.Do(func() {
		if statsAddr == "" {
			statsAddr = "127.0.0.1:5000"
		}
		go func() {
			log.Println(http.ListenAndServe(
				statsAddr,
				web.NewHandler(&multiStateProvider),
			))
		}()
	})

	var (
		sinks        []*syslog.Sink
		clusterSinks []*syslog.Sink
	)

	sink := &syslog.Sink{
		Addr:      addr,
		Name:      name,
		Namespace: namespace,
	}
	if tls != "" {
		var tlsConfig syslog.TLS
		err := json.Unmarshal([]byte(tls), &tlsConfig)
		if err != nil {
			log.Printf("[out_syslog] ERROR: Unable to unmarshal TLS config: %s", err)
			return output.FLB_ERROR
		}
		sink.TLS = &tlsConfig
	}
	if strings.ToLower(cluster) == "true" {
		clusterSinks = append(clusterSinks, sink)
	} else {
		sinks = append(sinks, sink)
	}
	out := syslog.NewOut(sinks, clusterSinks)
	multiStateProvider.Add(out)

	output.FLBPluginSetContext(plugin, unsafe.Pointer(&out))

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var (
		ret    int
		ts     interface{}
		record map[interface{}]interface{}
	)

	out := (*syslog.Out)(ctx)

	dec := output.NewDecoder(data, int(length))
	for {
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		var timestamp time.Time
		switch tts := ts.(type) {
		case output.FLBTime:
			timestamp = tts.Time
		case uint64:
			// From our observation, when ts is of type uint64 it appears to
			// be the amount of seconds since unix epoch.
			timestamp = time.Unix(int64(tts), 0)
		default:
			timestamp = time.Now()
		}

		out.Write(record, timestamp, C.GoString(tag))
	}

	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	// TODO: We should probably call conn.Close() for each sink connection
	return output.FLB_OK
}

func main() {
}
