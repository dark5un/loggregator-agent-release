package v2_test

import (
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	v2 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("BatchEnvelopeWriter", func() {
	It("processes each envelope before writing", func() {
		mockT := testhelpers.NewMockTesting()
		mockWriter := mocks.NewBatchWriter(mockT)

		// Capture the written batch for inspection
		var capturedBatch []*loggregator_v2.Envelope
		mockWriter.On("Write", mock.Anything).Run(func(args mock.Arguments) {
			capturedBatch = args.Get(0).([]*loggregator_v2.Envelope)
		}).Return(nil)

		tagger := v2.NewTagger(nil)
		ew := v2.NewBatchEnvelopeWriter(mockWriter, v2.NewCounterAggregator(tagger.TagEnvelope))
		envs := []*loggregator_v2.Envelope{
			buildEnvelopeWithCounter(10, "name-1", "origin-1"),
			buildEnvelopeWithCounter(14, "name-2", "origin-1"),
		}

		Expect(ew.Write(envs)).ToNot(HaveOccurred())

		// Verify the envelopes were passed to the writer
		Expect(mockWriter.Calls).To(Equal(1))
		Expect(capturedBatch).To(HaveLen(2))
		Expect(capturedBatch[0].GetCounter().GetTotal()).To(Equal(uint64(10)))
		Expect(capturedBatch[1].GetCounter().GetTotal()).To(Equal(uint64(14)))
	})
})

func buildEnvelopeWithCounter(total uint64, name, origin string) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		SourceId: origin,
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Total: total,
			},
		},
	}
}
