package web_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("State Handler", func() {
	It("responds with a 200", func() {
		fixedTime, _ := time.Parse(time.RFC3339, "2009-11-10T23:00:00Z")
		errorTime, _ := time.Parse(time.RFC3339, "2009-11-10T23:00:01Z")
		stats := syslogStater{
			s: []syslog.SinkState{
				{
					Name:               "sink-name",
					Namespace:          "ns1",
					LastSuccessfulSend: fixedTime,
					Error: &syslog.SinkError{
						Msg:       "some-error",
						Timestamp: errorTime,
					},
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
					"last_successful_send": "2009-11-10T23:00:00Z",
					"error": {
						"msg": "some-error",
						"timestamp": "2009-11-10T23:00:01Z"
					}
				}
			]
		`))
	})
})

type syslogStater struct {
	s []syslog.SinkState
}

func (ss syslogStater) SinkState() []syslog.SinkState {
	return ss.s
}
