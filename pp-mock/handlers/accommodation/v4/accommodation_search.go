// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"fmt"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v4/accommodationv4grpc"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ accommodationv4grpc.AccommodationSearchServiceServer = (*accommodationSearchV4Server)(nil)

type accommodationSearchV4Server struct{}

func NewAccommodationSearchServer() accommodationv4grpc.AccommodationSearchServiceServer {
	return &accommodationSearchV4Server{}
}

func (s *accommodationSearchV4Server) AccommodationSearch(_ context.Context, req *accommodationv4.AccommodationSearchRequest) (*accommodationv4.AccommodationSearchResponse, error) {
	now := time.Now()

	if !common.IsTravelPeriodAllowedV4WithTime(now, req.TravelPeriod) {
		return &accommodationv4.AccommodationSearchResponse{
			Response: &accommodationv4.AccommodationSearchResponse_ErrorResponse{
				ErrorResponse: &accommodationv4.AccommodationSearchErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, common.TravelPeriodErrorStr),
				},
			},
		}, nil
	}

	searchResults := []*accommodationv4.AccommodationSearchResult{}
	resultIDnum := uint32(0)
	validationPrices := []*state.UnifiedPrice{}

	// loop request queries
	filteredProps := filterExtendedPropertiesByGeoTreeLocation(mockdata.PropertiesV4, req.SearchParametersAccommodation.GetLocationGeoTree())
	filteredProps = filterExtendedPropertiesByProductCodes(filteredProps, req.SearchParametersAccommodation.GetProductCodes())
	filteredProps = filterExtendedPropertiesBySupplierCodes(filteredProps, req.SearchParametersAccommodation.GetSupplierCodes())

	startDate := req.TravelPeriod.GetStartDate()
	startDateTime := common.DateV4ToTime(startDate)
	startDateTimestamp := timestamppb.New(startDateTime)

	endDate := req.TravelPeriod.GetEndDate()
	endDateTime := common.DateV4ToTime(endDate)
	endDateTimestamp := timestamppb.New(endDateTime)

	duration := common.DaysBetweenDatesV4(endDate, startDate)

	// generate search result
	for _, prop := range filteredProps {
		for _, room := range prop.Rooms {
			unitPriceValue := common.DefaultPricePerNight * duration // we use the same value for different currencies, because it's mock and its fine if it will be different prices
			unit := &accommodationv4.Unit{
				SupplierRoomCode: room.SupplierCode,
				SupplierRoomName: room.SupplierName,
				OriginalRoomName: room.OriginalName,
				TravelPeriod: &typesv4.TravelPeriod{
					StartDate: &typesv4.Date{
						Year:  startDate.GetYear(),
						Month: startDate.GetMonth(),
						Day:   startDate.GetDay(),
					},
					EndDate: &typesv4.Date{
						Year:  endDate.GetYear(),
						Month: endDate.GetMonth(),
						Day:   endDate.GetDay(),
					},
				},
				TravellerIds: getTravellerIDs(req.Travellers),
				Beds:         room.Beds,
				PriceDetail: &typesv4.PriceDetail{
					Price: &typesv4.Price{
						Value:    fmt.Sprintf("%d", unitPriceValue),
						Decimals: 0, // we always return 0 decimals so we won't need to deal with different currencies decimals in mock
						Currency: common.CloneProto(req.SearchParameters.Currency),
					},
					ChargeType: typesv4.ChargeType_CHARGE_TYPE_PER_UNIT,
				},
				Services: []*typesv4.ServiceFact{{ // temporary placeholder
					Code: "MOCK",
					PriceDetail: &typesv4.PriceDetail{
						Price: &typesv4.Price{
							Value:    "100",
							Decimals: 0,
							Currency: common.CloneProto(req.SearchParameters.Currency),
						},
						ChargeType:  typesv4.ChargeType_CHARGE_TYPE_PER_PERSON,
						Description: "Temporary mock placeholder.",
					},
					AvailabilityType: typesv4.ServiceAvailabilityType_SERVICE_AVAILABILITY_TYPE_COMPULSORY,
					ChargeBasis:      typesv4.ChargeBasisType_CHARGE_BASIS_TYPE_ONCE,
				}}, // TODO evlekht@ use mockdata for services (not there yet)
				MealPlan: room.MealPlans[0],
				RatePlan: &typesv4.RatePlan{
					Code: "DS",
					Type: typesv4.RatePlanType_RATE_PLAN_TYPE_REGULAR,
				},
				RemainingUnits: 100, // hardcoded default value
				PropertyCode:   prop.Property.ProductCodes,
				SupplierCode:   prop.Property.SupplierCode,
			}

			cancelPenalties := []*typesv4.CancelPenalty{}
			if startDateTime.After(now.Add(common.FreeCancellationDuration)) {
				cancelPenalties = append(cancelPenalties, &typesv4.CancelPenalty{
					DatetimeRange: &typesv4.DateTimeRange{
						Start: timestamppb.New(now),
						End:   timestamppb.New(startDateTime.Add(-common.FreeCancellationDuration)),
					},
					Value: &typesv4.Price{
						Value:    "0", // 0% penalty
						Decimals: unit.PriceDetail.Price.Decimals,
						Currency: unit.PriceDetail.Price.Currency,
					},
					ValidForRatePlans: []string{unit.RatePlan.Code},
				})
			}

			penalty1StartSeconds := max(startDateTime.Add(-common.FreeCancellationDuration).Unix(), now.Unix())

			cancelPenalties = append(cancelPenalties,
				&typesv4.CancelPenalty{
					DatetimeRange: &typesv4.DateTimeRange{
						Start: &timestamppb.Timestamp{Seconds: penalty1StartSeconds},
						End:   timestamppb.New(startDateTime),
					},
					Value: &typesv4.Price{
						Value:    fmt.Sprintf("%d", unitPriceValue/10), // 10% penalty
						Decimals: unit.PriceDetail.Price.Decimals,
						Currency: unit.PriceDetail.Price.Currency,
					},
					ValidForRatePlans: []string{unit.RatePlan.Code},
				},
				&typesv4.CancelPenalty{
					DatetimeRange: &typesv4.DateTimeRange{
						Start: startDateTimestamp,
						End:   endDateTimestamp,
					},
					Value:             unit.PriceDetail.Price,
					ValidForRatePlans: []string{unit.RatePlan.Code},
				},
			)

			searchResults = append(searchResults, &accommodationv4.AccommodationSearchResult{
				ResultId: resultIDnum,
				TotalPrice: &typesv4.TotalPrice{
					Value: unit.PriceDetail.Price,
					CancelPolicy: &typesv4.CancelPolicy{
						PolicyType: &typesv4.CancelPolicy_ComplexCancelPenalties{
							ComplexCancelPenalties: &typesv4.ComplexCancelPenalties{
								CancelPenalties: cancelPenalties,
							},
						},
					},
				},
				Unit: unit,
				Bookability: &typesv4.Bookability{
					Type: typesv4.BookabilityType_BOOKABILITY_TYPE_AVAILABLE,
				},
			})

			validationPrice := state.PriceV4ToUnifiedPrice(unit.PriceDetail.Price)
			validationPrices = append(validationPrices, validationPrice)

			resultIDnum++
		}
	}

	resp := &accommodationv4.AccommodationSearchResponse{
		Response: &accommodationv4.AccommodationSearchResponse_SuccessResponse{
			SuccessResponse: &accommodationv4.AccommodationSearchSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				SearchId:   common.NewExpiringUUIDWithTime(now),
				Results:    searchResults,
				Travellers: req.Travellers,
			},
		},
	}

	if len(searchResults) == 0 {
		common.AddHeaderAlertV4(resp.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No results found")
	} else {
		state.GetStore().AddSearchResult(resp.GetSuccessResponse().SearchId.Id.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: resp.String(),
		})
	}

	return resp, nil
}

// Extracts traveller IDs from []*typesv4.BasicTraveller
func getTravellerIDs(travellers []*typesv4.BasicTraveller) []uint32 {
	ids := make([]uint32, len(travellers))
	for i := range travellers {
		ids[i] = travellers[i].TravellerId
	}
	return ids
}
