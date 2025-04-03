package v2_test

import (
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	v2 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BatchEnvelopeWriter", func() {
	var (
		ctrl       *gomock.Controller
		mockWriter *MockBatchWriter
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockWriter = NewMockBatchWriter(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("processes each envelope before writing", func() {
		envs := []*loggregator_v2.Envelope{
			buildCounterEnvelope(10, "name-1", "origin-1"),
			buildCounterEnvelope(14, "name-2", "origin-1"),
		}

		mockWriter.EXPECT().Write(envs).Return(nil)

		tagger := v2.NewTagger(nil)
		ew := v2.NewBatchEnvelopeWriter(mockWriter, v2.NewCounterAggregator(tagger.TagEnvelope))

		Expect(ew.Write(envs)).ToNot(HaveOccurred())
	})
})
