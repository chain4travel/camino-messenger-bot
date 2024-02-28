/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tracing

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

var _ Tracer = (*noopTracer)(nil)

type noopTracer struct {
	tp trace.TracerProvider
}

func NewNoOpTracer() (Tracer, error) {
	return &noopTracer{tp: trace.NewNoopTracerProvider()}, nil
}
func (n *noopTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return n.tp.Tracer("").Start(ctx, spanName, opts...)
}
func (n *noopTracer) Shutdown() error {
	return nil // nothing to do here
}
