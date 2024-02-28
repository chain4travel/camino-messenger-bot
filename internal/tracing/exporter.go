/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package tracing

import (
	"context"
	"fmt"
	"github.com/chain4travel/camino-messenger-bot/config"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

const exportTimeout = 10 * time.Second
const exporterInstantiationTimeout = 5 * time.Second

func newExporter(cfg *config.TracingConfig) (trace.SpanExporter, error) {

	var client otlptrace.Client
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		otlptracegrpc.WithTimeout(exportTimeout),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		creds, err := utils.LoadTLSCredentials(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load TLS keys: %s", err)
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(creds))
	}
	client = otlptracegrpc.NewClient(opts...)
	ctx, cancel := context.WithTimeout(context.Background(), exporterInstantiationTimeout)
	defer cancel()
	return otlptrace.New(ctx, client)
}
