// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"

	insurancev3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
)

var _ suite.Test = (*TestInsuranceV3)(nil)

func init() {
	Tests["InsuranceV3"] = &TestInsuranceV3{}
}

type TestInsuranceV3 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestInsuranceV3) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestInsuranceV3) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("InsuranceProductList", func(t *testing.T) {
		tt.testInsuranceProductList(ctx, t)
	})
	t.Run("InsuranceProductInfo", func(t *testing.T) {
		tt.testInsuranceProductInfo(ctx, t)
	})
	t.Run("Search", func(t *testing.T) {
		tt.testInsuranceSearch(ctx, t)
	})
}

func (tt *TestInsuranceV3) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.InsuranceProductListServiceV3,
		botGenerated.InsuranceProductInfoServiceV3,
		botGenerated.InsuranceSearchServiceV3,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.InsuranceProductListServiceV3, Fee: 100},
			{Name: botGenerated.InsuranceProductInfoServiceV3, Fee: 110},
			{Name: botGenerated.InsuranceSearchServiceV3, Fee: 120},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

func (tt *TestInsuranceV3) testInsuranceProductList(ctx context.Context, t *testing.T) {
	req := &insurancev3.InsuranceProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.InsuranceProductListServiceV3.InsuranceProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")
}

func (tt *TestInsuranceV3) testInsuranceProductInfo(ctx context.Context, t *testing.T) {
	req := &insurancev3.InsuranceProductInfoRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.InsuranceProductInfoServiceV3.InsuranceProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")
}

func (tt *TestInsuranceV3) testInsuranceSearch(ctx context.Context, t *testing.T) {
	req := &insurancev3.InsuranceSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.InsuranceSearchServiceV3.InsuranceSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")
}
