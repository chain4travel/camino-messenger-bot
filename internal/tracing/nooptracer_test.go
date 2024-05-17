/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/trace"
)

func TestNoopTraceIDForSpan(t *testing.T) {
	tracer, err := NewNoOpTracer()
	require.NoError(t, err)
	_, span := tracer.Start(context.Background(), "test")
	got := tracer.TraceIDForSpan(span)
	emptyTraceID := trace.TraceID{}
	require.NotEqual(t, emptyTraceID, got)
}
