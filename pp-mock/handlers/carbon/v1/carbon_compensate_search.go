// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/carbon/v1/carbonv1grpc"
	carbonv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/carbon/v1"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
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
	resultIDnum := int32(1)
	for _, query := range req.Queries {

		// Check if SearchParametersCarbon is missing
		if query.SearchParametersCarbon == nil {
			return &carbonv1.CarbonCompensateSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Mandatory field SearchParametersCarbon is missing",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		}

		// Check if CompensationType is not UNDEFINED
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
				var locAmount float32
				if accommodation.GetLocationCode() != nil {
					locAmount = mockdata.LocationCodeToAmountPerDay[accommodation.GetLocationCode().Code]
				} else {
					locAmount = 15.0
				}

				var amount float32
				if accommodation.GetPeriod() != nil {
					days := (float32(accommodation.GetPeriod().EndDatetime.Seconds) - float32(accommodation.GetPeriod().StartDatetime.Seconds)) / (24 * 60 * 60)
					amount = days * locAmount
				} else {
					// Default to 1 day if no period specified
					amount = locAmount
				}

				p := &carbonv1.CarbonCompensation{
					Id:        accommodation.Id,
					Reference: int32(resultIDnum), // TODO Reference is a string in request, but int32 in response
					Price: &typesv3.Price{
						Value:    fmt.Sprintf("%d", int(10*amount)),
						Decimals: 2,
					},
					Amount:     amount,
					ProposalId: "1234567890",
				}

				carbonSearchResults = append(carbonSearchResults, &carbonv1.CarbonSearchResult{
					CompensationPackage: []*carbonv1.CarbonCompensation{p},
					QueryId:             query.QueryId,
					ResultId:            resultIDnum, // TODO: make unique
					TotalPrice: &typesv3.Price{
						Value:    fmt.Sprintf("%d", int(100)),
						Decimals: 2,
						Currency: req.SearchParametersGeneric.Currency,
					},
				},
				)
				resultIDnum++
			}

			for _, transport := range query.GetTransport() {
				var totalTransportAmount float32

				// Get departure and arrival location codes
				var departureCodes, arrivalCodes []string

				if transport.From != nil && transport.From.Location != nil {
					if locationCodes, ok := transport.From.Location.(*transportv3.QueryTransitEventLocation_LocationCodes); ok {
						for _, code := range locationCodes.LocationCodes.Codes {
							departureCodes = append(departureCodes, code.Code)
						}
					}
				}

				if transport.To != nil && transport.To.Location != nil {
					if locationCodes, ok := transport.To.Location.(*transportv3.QueryTransitEventLocation_LocationCodes); ok {
						for _, code := range locationCodes.LocationCodes.Codes {
							arrivalCodes = append(arrivalCodes, code.Code)
						}
					}
				}

				// Check if from and to have equal length
				if len(departureCodes) != len(arrivalCodes) {
					log.Printf("Warning: From and To location codes have different lengths for transport query %d", transport.Id)
					continue
				}

				// Process all route pairs and sum up the amounts
				for i := 0; i < len(departureCodes); i++ {
					routeKey := fmt.Sprintf("%s-%s", departureCodes[i], arrivalCodes[i])

					// Get transport amount from mock data based on vehicle type
					var amount float32
					var exists bool

					switch transport.VehicleType {
					case "plane", "airplane", "aircraft":
						amount, exists = mockdata.PlaneRouteToAmount[routeKey]
					case "train", "railway", "rail":
						amount, exists = mockdata.TrainRouteToAmount[routeKey]
					default:
						// Default to plane mapping for unknown vehicle types
						amount, exists = mockdata.PlaneRouteToAmount[routeKey]
					}

					if exists {
						totalTransportAmount += amount
					} else {
						// Default amount if route not found - higher for planes, lower for trains
						if transport.VehicleType == "train" || transport.VehicleType == "railway" || transport.VehicleType == "rail" {
							totalTransportAmount += 5.0 // Lower default for trains
						} else {
							totalTransportAmount += 51.0 // Higher default for planes
						}
					}
				}

				price := float32(120 * totalTransportAmount) // e.g. 120 cents per kg CO2
				p := &carbonv1.CarbonCompensation{
					Id:        transport.Id,
					Reference: int32(resultIDnum),
					Price: &typesv3.Price{
						Value:    fmt.Sprintf("%d", int(price*1)),
						Decimals: 2,
						Currency: req.SearchParametersGeneric.Currency,
					},
					Amount:     totalTransportAmount * 1,
					ProposalId: "1234567890",
				}

				carbonSearchResults = append(carbonSearchResults, &carbonv1.CarbonSearchResult{
					CompensationPackage: []*carbonv1.CarbonCompensation{p},
					QueryId:             query.QueryId,
					ResultId:            resultIDnum,
					TotalPrice: &typesv3.Price{
						Value:    fmt.Sprintf("%d", int(price)),
						Decimals: 2,
						Currency: req.SearchParametersGeneric.Currency,
					},
				})
				resultIDnum++
			}
		}
	}

	response := &carbonv1.CarbonCompensateSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: carbonSearchResults,
	}

	if len(carbonSearchResults) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: fmt.Sprintf("No results found for search %v", req.Queries),
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}

		// Store search results for validation
		validationPrices := make([]*state.UnifiedPrice, len(carbonSearchResults))
		for i, result := range carbonSearchResults {
			validationPrices[i] = &state.UnifiedPrice{
				Price:                result.TotalPrice.Value,
				Decimals:             result.TotalPrice.Decimals,
				IsNative:             result.TotalPrice.Currency != nil && result.TotalPrice.Currency.GetNativeToken() != nil,
				IsoCurrencyEnum:      int32(result.TotalPrice.Currency.GetIsoCurrency()),
				TokenContractAddress: "", // Not used for carbon compensation
			}
		}

		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(carbonSearchResults),
			NumTravelers: 1, // Default for carbon compensation
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
		})
	}

	// Set gRPC headers
	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	response.Results = carbonSearchResults
	return response, nil
}
