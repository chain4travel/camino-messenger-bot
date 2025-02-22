// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	botGenerated "github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func protoMessageToJSON(tt *Test, message proto.Message) string {
	// Pretty-print using protojson.MarshalOptions
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonData, err := marshaler.Marshal(message)
	if err != nil {
		tt.logger.Errorf("Error marshalling: %v", err)
		return ""
	}
	// return the json data as string
	return string(jsonData)
}

func TestAccommodationV2Setup(ctx context.Context, t *testing.T, tt *Test) (*partnerplugin.PartnerPlugin, *bot.Bot, *bot.Bot) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.AccommodationProductListServiceV2))
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.AccommodationProductInfoServiceV2))
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.AccommodationSearchServiceV2))
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.ValidationServiceV2))
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.MintServiceV2))

	supplierPartnerPlugin := tt.CreatePartnerPlugin(ctx, t)

	supplierServices := []bot.CMService{
		{Name: botGenerated.AccommodationProductListServiceV2, Fee: 100},
		{Name: botGenerated.AccommodationProductInfoServiceV2, Fee: 110},
		{Name: botGenerated.AccommodationSearchServiceV2, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
	}

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot := tt.CreateBot(ctx, t, false, supplierPartnerPlugin, supplierServices)

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot := tt.CreateBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

func TestAccommodationProductListServiceV2(t *testing.T, tt *Test, distributorBot *bot.Bot, supplierBot *bot.Bot, ctx context.Context) {
	resp, err := distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&accommodationv2.AccommodationProductListRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		},
	)
	tt.logger.Debug("AccommodationProductListServiceV2.AccommodationProductList response:\n", protoMessageToJSON(tt, resp))

	require.NoError(t, err)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all 5 properties defined by the pp-mock
	require.Len(t, resp.Properties, 5, "unexpected number of properties in response")

	// Let's check if the first one is as expected
	require.NotEmpty(t, resp.Properties, "unexpected empty response properties")
	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, "HOTEL123456", resp.Properties[0].SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")
}

func TestAccommodationProductListServiceV2WithFilter(t *testing.T, tt *Test, distributorBot *bot.Bot, supplierBot *bot.Bot, ctx context.Context) {
	resp, err := distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&accommodationv2.AccommodationProductListRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			ModifiedAfter: &timestamppb.Timestamp{
				Seconds: 1710489050,
			},
		},
	)
	tt.logger.Debug("AccommodationProductListServiceV2.AccommodationProductList response:\n", protoMessageToJSON(tt, resp))

	require.NoError(t, err)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one property as only one is modified after the given timestamp
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Properties, "unexpected empty response properties")
	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, "HOTEL567890", resp.Properties[0].SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")
}

func TestAccommodationV2(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	_, supplierBot, distributorBot := TestAccommodationV2Setup(ctx, t, tt)

	TestAccommodationProductListServiceV2(t, tt, distributorBot, supplierBot, ctx)
	TestAccommodationProductListServiceV2WithFilter(t, tt, distributorBot, supplierBot, ctx)

}
