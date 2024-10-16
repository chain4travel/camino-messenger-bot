/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tracing

import (
	"context"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

const tracerProviderShutdownTimeout = exportTimeout + 5*time.Second

var _ Tracer = (*tracer)(nil)

type Tracer interface {
	trace.Tracer
	TraceIDForSpan(trace.Span) trace.TraceID // TraceIDForSpan returns the trace ID of the given span.
	Shutdown() error
}

type tracer struct {
	embedded.Tracer
	*sdktrace.TracerProvider
}

func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.TracerProvider.Tracer("").Start(ctx, spanName, opts...)
}

func (t *tracer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), tracerProviderShutdownTimeout)
	defer cancel()
	return t.TracerProvider.Shutdown(ctx)
}

func (t *tracer) TraceIDForSpan(span trace.Span) trace.TraceID {
	return span.SpanContext().TraceID()
}

func NewTracer(cfg config.TracingConfig, name string) (Tracer, error) {
	exporter, err := newExporter(&cfg)
	if err != nil {
		return nil, err
	}

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(name),
		)),
	)
	otel.SetTracerProvider(tp)

	return &tracer{
		TracerProvider: tp,
	}, nil
}
