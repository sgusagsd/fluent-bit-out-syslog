package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

type syslogStateProvider interface {
	SinkState() []syslog.SinkState
}

func NewHandler(p syslogStateProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s := p.SinkState()

		err := json.NewEncoder(w).Encode(&s)
		if err != nil {
			log.Printf("Error encoding state response: %s", err)
		}
	}
}
