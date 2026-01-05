// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestServiceV1              = "TestServiceV1"
	TestServiceV1Request  Type = TestServiceV1 + ".Request"
	TestServiceV1Response Type = TestServiceV1 + ".Response"
)

func TestMessageTypeToServiceName(t *testing.T) {
	tests := []struct {
		messageType Type
		expected    string
	}{
		{
			messageType: TestServiceV1Request,
			expected:    TestServiceV1,
		},
		{
			messageType: TestServiceV1Response,
			expected:    TestServiceV1,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.messageType), func(t *testing.T) {
			require.Equal(t, tt.expected, tt.messageType.ToServiceName())
		})
	}
}
