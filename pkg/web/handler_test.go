package web_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handler", func() {
	It("responds with a 200", func() {
		stats := syslogStater{
			stat: []syslog.Stat{
				{
					Name:          "sink-name",
					Namespace:     "ns1",
					LastSentNanos: 10,
					WriteError:    "error",
				},
			},
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		web.NewHandler(stats)(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(MatchJSON(`
			[
				{
					"name": "sink-name",
					"namespace": "ns1",
					"last_sent_nanos": 10,
					"write_error": "error"
				}
			]
		`))
	})
})

type syslogStater struct {
	stat []syslog.Stat
}

func (s syslogStater) Stats() []syslog.Stat {
	return s.stat
}
