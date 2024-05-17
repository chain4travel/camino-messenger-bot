/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTraceIDForSpan(t *testing.T) {
	tracer, err := NewNoOpTracer()
	require.NoError(t, err)
	_, span := tracer.Start(context.Background(), "test")
	got := tracer.TraceIDForSpan(span)
	require.NotEqual(t, span.SpanContext().TraceID(), got)
}
