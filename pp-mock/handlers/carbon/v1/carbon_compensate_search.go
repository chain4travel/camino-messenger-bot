// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/carbon/v1/carbonv1grpc"
	carbonv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/carbon/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

type carbonCompensateSearchV1Server struct {
	eventSender events.Sender
}

func NewCarbonCompensateSearchV1Server(eventSender events.Sender) carbonv1grpc.CarbonCompensateServiceServer {
	return &carbonCompensateSearchV1Server{eventSender: eventSender}
}

func (s *carbonCompensateSearchV1Server) CarbonCompensateSearch(ctx context.Context, req *carbonv1.CarbonCompensateSearchRequest) (*carbonv1.CarbonCompensateSearchResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

	fmt.Printf("Search generic params: %+v\n", req.SearchParametersGeneric)

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Carbon Compensate Search): %s", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &carbonv1.CarbonCompensateSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "No queries provided",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if req.SearchParametersGeneric == nil {
		return &carbonv1.CarbonCompensateSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field SearchParametersGeneric is missing",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}
	carbonSearchResults := []*carbonv1.CarbonSearchResult{}
	for _, query := range req.Queries {
		if query.SearchParametersCarbon.CompensationType == carbonv1.CompensationType_COMPENSATION_TYPE_UNSPECIFIED {
			return &carbonv1.CarbonCompensateSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Mandatory field CompensationType is missing",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		} else if query.SearchParametersCarbon.CompensationType == carbonv1.CompensationType_COMPENSATION_TYPE_PURCHASE_TREE {
			return &carbonv1.CarbonCompensateSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Not offering tree purchase",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		} else if query.SearchParametersCarbon.CompensationType == carbonv1.CompensationType_COMPENSATION_TYPE_CO2_DEBITS {
			for _, accommodation := range query.GetAccommodation() {

				locAmount := mockdata.LocationCodeToAmountPerDay[accommodation.GetLocationCode().Code]
				days := (float32(accommodation.GetPeriod().EndDatetime.Seconds) - float32(accommodation.GetPeriod().StartDatetime.Seconds)) / (24 * 60 * 60)
				amount := days * locAmount

				p := &carbonv1.CarbonCompensation{
					Price: &typesv3.Price{
						Value:    fmt.Sprintf("%d", int(10*amount)),
						Decimals: 2,
					},
					Amount: amount,
				}

				carbonSearchResults = append(carbonSearchResults, &carbonv1.CarbonSearchResult{
					CompensationPackage: []*carbonv1.CarbonCompensation{p},
					QueryId:             query.QueryId,
					ResultId:            1234,
				},
				)
			}
		}
	}

	return &carbonv1.CarbonCompensateSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: carbonSearchResults,
	}, nil
}
