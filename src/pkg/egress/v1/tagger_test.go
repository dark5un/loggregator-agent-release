package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tagger", func() {
	It("adds the deployment tag", func() {
		mockT := testhelpers.NewMockTesting()
		mockWriter := mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()

		tagger := egress.NewTagger("my-dep", "my-job", "my-index", "my-ip", mockWriter)

		env := basicHttpStartStopEnvelope()
		tagger.Write(env)

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		Expect(mockWriter.Envelopes[0].GetDeployment()).To(Equal("my-dep"))
	})

	It("adds the job tag", func() {
		mockT := testhelpers.NewMockTesting()
		mockWriter := mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()

		tagger := egress.NewTagger("my-dep", "my-job", "my-index", "my-ip", mockWriter)

		env := basicHttpStartStopEnvelope()
		tagger.Write(env)

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		Expect(mockWriter.Envelopes[0].GetJob()).To(Equal("my-job"))
	})

	It("adds the index tag", func() {
		mockT := testhelpers.NewMockTesting()
		mockWriter := mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()

		tagger := egress.NewTagger("my-dep", "my-job", "my-index", "my-ip", mockWriter)

		env := basicHttpStartStopEnvelope()
		tagger.Write(env)

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		Expect(mockWriter.Envelopes[0].GetIndex()).To(Equal("my-index"))
	})

	It("adds the IP tag", func() {
		mockT := testhelpers.NewMockTesting()
		mockWriter := mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()

		tagger := egress.NewTagger("my-dep", "my-job", "my-index", "my-ip", mockWriter)

		env := basicHttpStartStopEnvelope()
		tagger.Write(env)

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		Expect(mockWriter.Envelopes[0].GetIp()).To(Equal("my-ip"))
	})
})

func basicHttpStartStopEnvelope() *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("some-origin"),
		EventType: events.Envelope_HttpStartStop.Enum(),
	}
}
