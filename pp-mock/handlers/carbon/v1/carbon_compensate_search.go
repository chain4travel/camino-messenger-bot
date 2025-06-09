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
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
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
							Type: typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
				},
			}, nil

		} else if query.SearchParametersCarbon.CompensationType == carbonv1.CompensationType_COMPENSATION_TYPE_CO2_DEBITS {


			

		}

	}
	return nil, nil
}
