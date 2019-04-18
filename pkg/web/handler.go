package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

type MultiStateProvider struct {
	providers []SyslogStateProvider
}

func (msp *MultiStateProvider) Add(sp SyslogStateProvider) {
	msp.providers = append(msp.providers, sp)
}

func (msp *MultiStateProvider) SinkState() []syslog.SinkState {
	var result []syslog.SinkState
	for _, provider := range msp.providers {
		result = append(result, provider.SinkState()...)
	}
	return result
}

type SyslogStateProvider interface {
	SinkState() []syslog.SinkState
}

func NewHandler(p SyslogStateProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s := p.SinkState()

		err := json.NewEncoder(w).Encode(&s)
		if err != nil {
			log.Printf("Error encoding state response: %s", err)
		}
	}
}
