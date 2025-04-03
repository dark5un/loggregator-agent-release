package bindings_test

import (
	"errors"
	"log"
	"net"

	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/ingress/bindings"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type SpyBindingReader struct {
	bindings []syslog.Binding
	err      error
}

func (s *SpyBindingReader) FetchBindings() ([]syslog.Binding, error) {
	return s.bindings, s.err
}

func (s *SpyBindingReader) DrainLimit() int {
	return 0
}

var _ = Describe("FilteredBindingFetcher", func() {
	var (
		mockCtrl    *gomock.Controller
		mockChecker *MockIPChecker
		fetcher     *bindings.FilteredBindingFetcher
		metrics     *metricsHelpers.SpyMetricsRegistry
		logger      *log.Logger
		reader      *SpyBindingReader
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockChecker = NewMockIPChecker(mockCtrl)
		metrics = metricsHelpers.NewMetricsRegistry()
		logger = log.New(GinkgoWriter, "", 0)
		reader = &SpyBindingReader{}
		fetcher = bindings.NewFilteredBindingFetcher(mockChecker, reader, metrics, true, logger)
	})

	AfterEach(func() {
		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	It("checks bindings against blacklist", func() {
		mockChecker.EXPECT().ResolveAddr("some-addr").Return(net.ParseIP("127.0.0.1"), nil)
		mockChecker.EXPECT().CheckBlacklist(net.ParseIP("127.0.0.1")).Return(nil)

		err := fetcher.CheckBindings([]string{"some-addr"})
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns error when resolving address fails", func() {
		mockChecker.EXPECT().ResolveAddr("some-addr").Return(nil, errors.New("some-error"))

		err := fetcher.CheckBindings([]string{"some-addr"})
		Expect(err).To(MatchError("failed to resolve address some-addr: some-error"))
	})

	It("returns error when checking blacklist fails", func() {
		mockChecker.EXPECT().ResolveAddr("some-addr").Return(net.ParseIP("127.0.0.1"), nil)
		mockChecker.EXPECT().CheckBlacklist(net.ParseIP("127.0.0.1")).Return(errors.New("some-error"))

		err := fetcher.CheckBindings([]string{"some-addr"})
		Expect(err).To(MatchError("failed to check blacklist for some-addr: some-error"))
	})

	Context("when fetching bindings", func() {
		It("returns error from binding reader", func() {
			reader.err = errors.New("some-error")
			_, err := fetcher.FetchBindings()
			Expect(err).To(MatchError("some-error"))
		})

		It("filters invalid URLs", func() {
			reader.bindings = []syslog.Binding{
				{AppId: "app-id", Hostname: "we.dont.care", Drain: syslog.Drain{Url: "://"}},
			}
			bindings, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())
			Expect(bindings).To(BeEmpty())
		})

		It("filters URLs with invalid schemes", func() {
			reader.bindings = []syslog.Binding{
				{AppId: "app-id", Hostname: "we.dont.care", Drain: syslog.Drain{Url: "foo://example.com"}},
			}
			bindings, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())
			Expect(bindings).To(BeEmpty())
		})

		It("filters URLs with no host", func() {
			reader.bindings = []syslog.Binding{
				{AppId: "app-id", Hostname: "we.dont.care", Drain: syslog.Drain{Url: "syslog:///"}},
			}
			bindings, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())
			Expect(bindings).To(BeEmpty())
		})

		It("filters blacklisted hosts", func() {
			reader.bindings = []syslog.Binding{
				{AppId: "app-id", Hostname: "we.dont.care", Drain: syslog.Drain{Url: "syslog://example.com"}},
			}
			mockChecker.EXPECT().ResolveAddr("example.com").Return(net.ParseIP("127.0.0.1"), nil)
			mockChecker.EXPECT().CheckBlacklist(net.ParseIP("127.0.0.1")).Return(errors.New("blacklisted"))

			bindings, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())
			Expect(bindings).To(BeEmpty())
		})

		It("allows valid bindings", func() {
			reader.bindings = []syslog.Binding{
				{AppId: "app-id", Hostname: "we.dont.care", Drain: syslog.Drain{Url: "syslog://example.com"}},
			}
			mockChecker.EXPECT().ResolveAddr("example.com").Return(net.ParseIP("127.0.0.1"), nil)
			mockChecker.EXPECT().CheckBlacklist(net.ParseIP("127.0.0.1")).Return(nil)

			bindings, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())
			Expect(bindings).To(Equal(reader.bindings))
		})
	})
})
