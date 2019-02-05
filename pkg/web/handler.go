package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

type syslogStatProvider interface {
	Stats() []syslog.Stat
}

func NewHandler(p syslogStatProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats := p.Stats()

		err := json.NewEncoder(w).Encode(&stats)
		if err != nil {
			log.Printf("Error encoding stat response: %s", err)
		}
	}
}
