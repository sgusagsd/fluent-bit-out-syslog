package syslog_test

import (
	"time"

	"code.cloudfoundry.org/rfc5424"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

var _ = Describe("Out", func() {
	It("adds structured data to syslog message", func() {
		spySink := newSpySink()
		defer spySink.stop()
		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "kube-system",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("etcd-minikube"),
				"namespace_name": []byte("kube-system"),
				"host":           []byte("some-host"),
				"container_name": []byte("etcd"),
			},
		}

		err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

		Expect(err).ToNot(HaveOccurred())
		spySink.expectReceivedWithSD(
			[]rfc5424.StructuredData{
				{
					ID: "kubernetes@47450",
					Parameters: []rfc5424.SDParam{
						{
							Name:  "namespace_name",
							Value: "kube-system",
						},
						{
							Name:  "object_name",
							Value: "etcd-minikube",
						},
						{
							Name:  "container_name",
							Value: "etcd",
						},
					},
				},
			},
		)
	})

	It("includes event in the app name if set on the record", func() {
		spySink := newSpySink()
		defer spySink.stop()
		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
				"pod_name":       []byte("pod-name"),
				"container_name": []byte("container-name"),
			},
		}

		err := out.Write(record, time.Unix(0, 0).UTC(), "k8s.event")

		Expect(err).ToNot(HaveOccurred())
		spySink.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - k8s.event/ns1/pod-name/container-name - - [kubernetes@47450 namespace_name="ns1" object_name="pod-name" container_name="container-name"] some-log` + "\n",
		)
	})

	It("doesn't return an error if it fails to write to one of the sinks in a namespace", func() {
		spySink1 := newSpySink()
		spySink1.stop()
		spySink2 := newSpySink()
		defer spySink2.stop()

		s1 := &syslog.Sink{
			Addr:      spySink1.url(),
			Namespace: "ns1",
		}
		s2 := &syslog.Sink{
			Addr:      spySink2.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

		r := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}
		err := out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink2.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
		)
	})

	It("returns an error if all sinks fail to write successfully", func() {
		spySink1 := newSpySink()
		spySink1.stop()
		spySink2 := newSpySink()
		spySink2.stop()

		s1 := &syslog.Sink{
			Addr:      spySink1.url(),
			Namespace: "ns1",
		}
		s2 := &syslog.Sink{
			Addr:      spySink2.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut([]*syslog.Sink{s1}, []*syslog.Sink{s2})

		r := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}
		err := out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).To(HaveOccurred())
	})

	It("filters messages based on namespace_name in kubernetes metadata", func() {
		spySink := newSpySink()
		defer spySink.stop()

		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "kube-system",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("etcd-minikube"),
				"namespace_name": []byte("kube-system"),
				"host":           []byte("some-host"),
				"container_name": []byte("etcd"),
			},
		}

		err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/kube-system/etcd-minikube/etcd - - [kubernetes@47450 namespace_name="kube-system" object_name="etcd-minikube" container_name="etcd"] some-log` + "\n",
		)
	})

	It("drops messages with unconfigured namespaces", func() {
		spySink := newSpySink()
		defer spySink.stop()

		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "test-namespace",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		r1 := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("etcd-minikube"),
				"namespace_name": []byte("kube-system"),
				"host":           []byte("some-host"),
				"container_name": []byte("etcd"),
			},
		}
		r2 := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("etcd-minikube"),
				"namespace_name": []byte("test-namespace"),
				"host":           []byte("some-host"),
				"container_name": []byte("etcd"),
			},
		}

		err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())
		err = out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink.expectReceivedOnly(
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/test-namespace/etcd-minikube/etcd - - [kubernetes@47450 namespace_name="test-namespace" object_name="etcd-minikube" container_name="etcd"] some-log` + "\n",
		)
	})

	It("truncates the app name if there is too much information", func() {
		spySink := newSpySink()
		defer spySink.stop()

		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "namespace-name-very-long",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"log": []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("pod-name"),
				"namespace_name": []byte("namespace-name-very-long"),
				"host":           []byte("some-host"),
				"container_name": []byte("container-name-very-long"),
			},
		}

		err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/namespace-name-very-long/pod-name/contai - - [kubernetes@47450 namespace_name="namespace-name-very-long" object_name="pod-name" container_name="container-name-very-long"] some-log` + "\n",
		)
	})

	It("doesn't add a newline if one already exists in the message", func() {
		spySink := newSpySink()
		defer spySink.stop()

		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "namespace",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"log": []byte("some-log\n"),
			"kubernetes": map[interface{}]interface{}{
				"pod_name":       []byte("pod-name"),
				"namespace_name": []byte("namespace"),
				"host":           []byte("some-host"),
				"container_name": []byte("container"),
			},
		}

		err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

		Expect(err).ToNot(HaveOccurred())
		spySink.expectReceivedOnly(
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/namespace/pod-name/container - - [kubernetes@47450 namespace_name="namespace" object_name="pod-name" container_name="container"] some-log` + "\n",
		)
	})

	It("filters messages to multiple sinks for a namespace", func() {
		spySink1 := newSpySink()
		defer spySink1.stop()
		spySink2 := newTLSSpySink()
		defer spySink2.stop()

		s1 := &syslog.Sink{
			Addr:      spySink1.url(),
			Namespace: "ns1",
		}
		s2 := &syslog.Sink{
			Addr:      spySink2.url(),
			Namespace: "ns1",
			TLS: &syslog.TLS{
				InsecureSkipVerify: true,
			},
		}
		out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

		r1 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}
		go func() {
			defer GinkgoRecover()
			err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())
		}()

		spySink1.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
		)
		spySink2.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
		)
	})

	It("filters messages to multiple namespaces", func() {
		spySink1 := newSpySink()
		defer spySink1.stop()
		spySink2 := newSpySink()
		defer spySink2.stop()

		s1 := &syslog.Sink{
			Addr:      spySink1.url(),
			Namespace: "ns1",
		}
		s2 := &syslog.Sink{
			Addr:      spySink2.url(),
			Namespace: "ns2",
		}
		out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

		r1 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}
		r2 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns2"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns2"),
			},
		}
		err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())
		err = out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink1.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
		)
		spySink2.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2` + "\n",
		)
	})

	It("sends all messages to all cluster sinks", func() {
		spySink1 := newSpySink()
		defer spySink1.stop()
		spyClusterSink1 := newSpySink()
		defer spyClusterSink1.stop()
		spyClusterSink2 := newSpySink()
		defer spyClusterSink2.stop()

		s1 := &syslog.Sink{
			Addr:      spySink1.url(),
			Namespace: "ns1",
		}
		cs1 := &syslog.Sink{
			Addr: spyClusterSink1.url(),
		}
		cs2 := &syslog.Sink{
			Addr: spyClusterSink2.url(),
		}
		out := syslog.NewOut(
			[]*syslog.Sink{s1},
			[]*syslog.Sink{cs1, cs2},
		)

		r1 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}
		r2 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns2"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns2"),
			},
		}
		err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())
		err = out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
		Expect(err).ToNot(HaveOccurred())

		spySink1.expectReceivedOnly(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
		)
		spyClusterSink1.expectReceivedOnly(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1`+"\n",
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2`+"\n",
		)
		spyClusterSink2.expectReceivedOnly(
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1`+"\n",
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2`+"\n",
		)
	})

	DescribeTable(
		"sends no data when record excludes pertinent info",
		func(record map[interface{}]interface{}) {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			done := make(chan struct{})
			go func() {
				_, _ = spySink.lis.Accept()
				close(done)
			}()

			Consistently(done).ShouldNot(BeClosed())
		},
		Entry(
			"has no k8s map",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
			},
		),
		Entry(
			"has k8s map of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[string][]byte{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
		Entry(
			"has no namespace name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
		Entry(
			"has namespace of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []int{1, 2, 3, 4},
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
	)

	DescribeTable(
		"prints even with some missing data",
		func(record map[interface{}]interface{}, message string) {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			spySink.expectReceived(message)
		},
		Entry(
			"no log message",
			map[interface{}]interface{}{
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] `+"\n",
		),
		Entry(
			"log message is of different type",
			map[interface{}]interface{}{
				"log": []int{1, 2, 3, 4},
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] `+"\n",
		),
		Entry(
			"log message key is of different type",
			map[interface{}]interface{}{
				5: []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] `+"\n",
		),
		Entry(
			"no host",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"host key is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					1:                []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"host is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []int{1, 2, 3, 4},
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"no container name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/ - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name=""] some-log`+"\n",
		),
		Entry(
			"container name is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []int{1, 2, 3, 4},
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/ - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name=""] some-log`+"\n",
		),
		Entry(
			"no pod name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns//some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"pod name is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []int{1, 2, 3, 4},
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns//some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name="some-container"] some-log`+"\n",
		),
	)

	Context("TCP", func() {
		It("eventually connects to a failing syslog sink", func() {
			spySink := newSpySink()
			spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).To(HaveOccurred())

			// bring the sink back to life
			spySink = newSpySink(spySink.url())

			err = out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)
		})

		It("doesn't reconnect if connection already established", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}
			err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			err = out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				_, _ = spySink.lis.Accept()
			}()
			Consistently(done).ShouldNot(BeClosed())
		})

		It("reconnects if previous connection went away", func() {
			spySink := newSpySink()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())
			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			spySink.stop()
			spySink = newSpySink(spySink.url())

			r2 := map[interface{}]interface{}{
				"log": []byte("some-log-message-2"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			f := func() error {
				return out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
			}
			Eventually(f).Should(HaveOccurred())

			err = out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).ToNot(HaveOccurred())

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message-2` + "\n",
			)
		})

	})

	Context("TLS", func() {
		It("eventually connects to a failing syslog sink", func() {
			spySink := newTLSSpySink()
			spySink.stop()
			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			goOn := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).To(HaveOccurred())
				close(goOn)
			}()

			<-goOn
			// bring the sink back to life
			spySink = newTLSSpySink(spySink.url())

			go func() {
				defer GinkgoRecover()
				err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
			}()

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)
		})

		It("doesn't reconnect if connection already established", func() {
			spySink := newTLSSpySink()
			defer spySink.stop()
			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{s}, nil)

			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			go func() {
				defer GinkgoRecover()
				err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
			}()

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			goOn := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				err := out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
				close(goOn)
			}()
			<-goOn
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				_, _ = spySink.lis.Accept()
			}()
			Consistently(done).ShouldNot(BeClosed())
		})

		It("reconnects if previous connection went away", func() {
			spySink := newTLSSpySink()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			// TLS will block on waiting for handshake so the write needs
			// to occur in a separate go routine
			go func() {
				defer GinkgoRecover()
				err := out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
			}()
			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			spySink.stop()
			spySink = newTLSSpySink(spySink.url())

			r2 := map[interface{}]interface{}{
				"log": []byte("some-log-message-2"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			f := func() error {
				return out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
			}
			Eventually(f).Should(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				err := out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
			}()

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message-2` + "\n",
			)
		})

		It("writes messages via syslog-tls", func() {
			spySink := newTLSSpySink()
			defer spySink.stop()

			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}

			out := syslog.NewOut([]*syslog.Sink{s}, nil)
			r := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
				},
			}

			// TLS will block on waiting for handshake so the write needs
			// to occur in a separate go routine
			go func() {
				defer GinkgoRecover()
				err := out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
				Expect(err).ToNot(HaveOccurred())
			}()

			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns// - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name=""] some-log` + "\n",
			)
		})

		It("fails when connecting to non TLS endpoint", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}

			out := syslog.NewOut(
				[]*syslog.Sink{s},
				nil,
				syslog.WithDialTimeout(time.Millisecond),
			)
			r := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
				},
			}

			err := out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
			Expect(err).To(HaveOccurred())
		})
	})
})

// This is an example of a message that is sent to the syslog output plugin.
// It is here for documentation purposes.
var _ = map[interface{}]interface{}{
	"log":    []byte("log data"),
	"stream": []byte("stdout"),
	"time":   []byte("2018-07-16T17:47:16.61514406Z"),
	"kubernetes": map[interface{}]interface{}{
		"labels": map[interface{}]interface{}{
			"component":                     []byte("kube-addon-manager"),
			"kubernetes.io/minikube-addons": []byte("addon-manager"),
			"version":                       []byte("v8.6"),
		},
		"annotations": map[interface{}]interface{}{
			"kubernetes.io/config.hash":   []byte{},
			"kubernetes.io/config.mirror": []byte{},
			"kubernetes.io/config.seen":   []byte{},
			"kubernetes.io/config.source": []byte{},
		},
		"host":           []byte("minikube"),
		"container_name": []byte("kube-addon-manager"),
		"docker_id":      []byte("some-hash"),
		"pod_name":       []byte("kube-addon-manager-minikube"),
		"namespace_name": []byte("kube-system"),
		"pod_id":         []byte("some-hash"),
	},
}
