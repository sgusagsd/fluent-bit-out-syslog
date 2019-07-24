package main

import (
	"C"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/web"
)

var (
	out *syslog.Out
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(
		ctx,
		"syslog",
		"syslog output plugin that follows RFC 5424",
	)
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {
	s := output.FLBPluginConfigKey(ctx, "sinks")
	cs := output.FLBPluginConfigKey(ctx, "clustersinks")
	sanitizeHost := output.FLBPluginConfigKey(plugin, "sanitizehost")
	if s == "" && cs == "" {
		log.Println("[out_syslog] ERROR: Sinks or ClusterSinks need to be configured")
		return output.FLB_ERROR
	}

	log.Println("[out_syslog] sinks =", s)
	log.Println("[out_syslog] cluster sinks =", cs)

	var (
		sinks        []*syslog.Sink
		clusterSinks []*syslog.Sink
	)

	if len(s) != 0 {
		err := json.Unmarshal([]byte(s), &sinks)
		if err != nil {
			log.Printf("[out_syslog] ERROR: unable to unmarshal sinks: %s", err)
			return output.FLB_ERROR
		}
	}
	if len(cs) != 0 {
		err := json.Unmarshal([]byte(cs), &clusterSinks)
		if err != nil {
			log.Printf("[out_syslog] ERROR: unable to unmarshal cluster sinks: %s", err)
			return output.FLB_ERROR
		}
	}

	if len(sinks)+len(clusterSinks) == 0 {
		log.Println("[out_syslog] ERROR: require at least one sink or cluster sink")
		return output.FLB_ERROR
	}

	// Defaults to true so that plugin conforms better with rfc5424#section-6.2.4
	sanitize := true
	if len(sanitizeHost) != 0 {
		var err error
		sanitize, err = strconv.ParseBool(sanitizeHost)
		if err != nil {
			log.Printf("[out_syslog] ERROR: Unable to parse SanitizeHost: %s", err)
			return output.FLB_ERROR
		}
	}
	out := syslog.NewOut(
		sinks,
		clusterSinks,
		syslog.WithSanitizeHost(sanitize),
	)

	statsAddr := output.FLBPluginConfigKey(ctx, "statsaddr")
	if statsAddr == "" {
		statsAddr = "127.0.0.1:5000"
	}
	go func() {
		log.Println(http.ListenAndServe(statsAddr, web.NewHandler(out)))
	}()

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var (
		ret    int
		ts     interface{}
		record map[interface{}]interface{}
	)

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
