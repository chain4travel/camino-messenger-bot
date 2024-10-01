package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestServiceV1                     = "TestServiceV1"
	TestServiceV1Request  MessageType = TestServiceV1 + ".Request"
	TestServiceV2Response MessageType = TestServiceV1 + ".Response"
)

func TestMessageTypeToServiceName(t *testing.T) {
	tests := []struct {
		messageType MessageType
		expected    string
	}{
		{
			messageType: TestServiceV1Request,
			expected:    TestServiceV1,
		},
		{

			messageType: TestServiceV2Response,
			expected:    TestServiceV1,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.messageType), func(t *testing.T) {
			require.Equal(t, tt.expected, tt.messageType.ToServiceName())
		})
	}
}
