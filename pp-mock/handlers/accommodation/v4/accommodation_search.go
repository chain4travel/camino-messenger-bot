// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var _ accommodationv4grpc.AccommodationSearchServiceServer = (*accommodationSearchV3Server)(nil)

type accommodationSearchV3Server struct{}

func NewAccommodationSearchServer() accommodationv4grpc.AccommodationSearchServiceServer {
	return &accommodationSearchV3Server{}
}

func (s *accommodationSearchV3Server) AccommodationSearch(_ context.Context, req *accommodationv4.AccommodationSearchRequest) (*accommodationv4.AccommodationSearchResponse, error) {
	fmt.Printf("Search params: %+v\n", req.SearchParameters)
	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &accommodationv4.AccommodationSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "No queries provided",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// check if SearchParameters is nil or if Currency is nil
	if req.SearchParameters == nil || req.SearchParameters.Currency == nil {
		return &accommodationv4.AccommodationSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Mandatory field SearchParameters.Currency is missing",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// loop queries and check if there is travel period
	for _, query := range req.Queries {
		if query.TravelPeriod == nil {
			return &accommodationv4.AccommodationSearchResponse{
				Header: &typesv4.ResponseHeader{
					BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
					Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv4.Alert{{
						Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for accommodations (with limits of start/end values of now() / now() + 60 days)",
						Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		}

		if !common.IsTravelPeriodAllowedV4(query.TravelPeriod) {
			return &accommodationv4.AccommodationSearchResponse{
				Header: &typesv4.ResponseHeader{
					BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
					Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv4.Alert{{
						Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
						Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		}
	}

	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	unsupportedResp := &accommodationv4.AccommodationSearchResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv4.Alert{{
				Message: "Unsupported: Traveller definitions must be identical in all queries",
				Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
	for queryIndex, query := range req.Queries {
		for queryIndex2, query2 := range req.Queries {
			if queryIndex != queryIndex2 {
				travellersA := query.GetTravellers()
				travellersB := query2.GetTravellers()

				if len(travellersA) != len(travellersB) {
					return unsupportedResp, nil
				}

				for i, travellerA := range travellersA {
					travellerB := travellersB[i]
					if !proto.Equal(travellerA, travellerB) {
						return unsupportedResp, nil
					}
				}
			}
		}
	}

	searchResults := []*accommodationv4.AccommodationSearchResult{}
	resultIDnum := uint32(1)
	validationPrices := []*state.UnifiedPrice{}

	// loop request queries
	for _, query := range req.Queries {
		filteredProps := filterExtendedPropertiesByGeoTreeLocation(mockdata.PropertiesV4, query.SearchParametersAccommodation.GetLocationGeoTree())
		filteredProps = filterExtendedPropertiesByProductCodes(filteredProps, query.SearchParametersAccommodation.GetProductCodes())
		filteredProps = filterExtendedPropertiesBySupplierCodes(filteredProps, query.SearchParametersAccommodation.GetSupplierCodes())

		// extract the duration of the travel period in days
		duration := common.DaysBetweenDatesV4(query.TravelPeriod.GetEndDate(), query.TravelPeriod.GetStartDate())

		// generate search result
		for _, prop := range filteredProps {
			units := []*accommodationv4.Unit{}
			// loop all rooms
			for _, room := range prop.Rooms {
				units = append(units, &accommodationv4.Unit{
					Type:             accommodationv4.UnitType(prop.Property.CategoryUnit),
					SupplierRoomCode: room.SupplierCode,
					SupplierRoomName: room.SupplierName,
					OriginalRoomName: room.OriginalName,
					TravelPeriod: &typesv4.TravelPeriod{
						StartDate: &typesv4.Date{
							Year:  query.TravelPeriod.GetStartDate().GetYear(),
							Month: query.TravelPeriod.GetStartDate().GetMonth(),
							Day:   query.TravelPeriod.GetStartDate().GetDay(),
						},
						EndDate: &typesv4.Date{
							Year:  query.TravelPeriod.GetEndDate().GetYear(),
							Month: query.TravelPeriod.GetEndDate().GetMonth(),
							Day:   query.TravelPeriod.GetEndDate().GetDay(),
						},
					},
					TravellerIds: getTravellerIDs(query.Travellers),
					Beds:         room.Beds,
					PriceDetail: &typesv4.PriceDetail{
						Price: &typesv4.Price{
							Value:    common.DefaultPricePerNightStr,
							Decimals: common.DefaultPricePerNightDecimals,
							Currency: common.CloneProto(req.SearchParameters.Currency),
						},
						Description: "price per night",
					},
					Services:       []*typesv4.ServiceFact{},
					MealPlanCode:   &typesv4.MealPlan{},
					RatePlan:       &typesv4.RatePlan{},
					RateRule:       &typesv4.RateRule{},
					RemainingUnits: 0,
					PropertyCode:   &typesv4.ProductCode{},
					SupplierCode:   prop.Property.SupplierCode,
					Remarks:        "",
				})
			}

			searchPrice := &typesv4.Price{
				Value:    fmt.Sprintf("%d", common.DefaultPricePerNight*duration),
				Decimals: common.DefaultPricePerNightDecimals,
				Currency: common.CloneProto(req.SearchParameters.Currency),
			}
			searchResults = append(searchResults, &accommodationv4.AccommodationSearchResult{
				ResultId: resultIDnum,
				QueryId:  query.QueryId,
				TotalPrice: &typesv4.TotalPrice{
					Value: searchPrice,
				},
				Units: units,
			})

			validationPrice := state.PriceV4ToUnifiedPrice(searchPrice)
			validationPrices = append(validationPrices, validationPrice)

			resultIDnum++
		}
	}

	response := &accommodationv4.AccommodationSearchResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: searchResults,
	}

	if len(searchResults) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: fmt.Sprintf("No results found for search %v", req.Queries),
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv4.SearchResponseMetadata{
			SearchId: &typesv4.UUID{Value: uuid.New().String()},
		}
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Queries[0].Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
			SeatMapIndex: -1,
		})
	}

	return response, nil
}

// Extracts traveller IDs from []*typesv4.BasicTraveller
func getTravellerIDs(travellers []*typesv4.BasicTraveller) []uint32 {
	ids := make([]uint32, len(travellers))
	for i := range travellers {
		ids[i] = travellers[i].TravellerId
	}
	return ids
}
