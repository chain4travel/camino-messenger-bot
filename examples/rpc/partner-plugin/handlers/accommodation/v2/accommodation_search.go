package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/cache"
	mock_data "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/data/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationSearchServiceServer = (*AccommodationSearchV2Server)(nil)

type AccommodationSearchV2Server struct{}

func (*AccommodationSearchV2Server) AccommodationSearch(ctx context.Context, req *accommodationv2.AccommodationSearchRequest) (*accommodationv2.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}

	var search_generic_params = req.SearchParametersGeneric
	// print params
	fmt.Printf("Search generic params: %+v\n", search_generic_params)

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Search): %s", md.RequestID)

	// load mock data
	properties := mock_data.LoadPropertiesMockData()

	// log
	fmt.Printf("properties: %+v\n", properties)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &accommodationv2.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{
					{
						Message: "No queries provided",
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}

	searchResults := []*accommodationv2.AccommodationSearchResult{}
	available_properties := []*accommodationv2.PropertyExtendedInfo{}
	// loop request queries
	for _, query := range req.Queries {
		props := make([]*accommodationv2.PropertyExtendedInfo, len(properties))
		for i := range properties {
			props[i] = &properties[i]
		}

		// get filtered properties
		var filtered_props = filterPropertiesByGeoTreeLocation(props, query.SearchParametersAccommodation.GetLocationGeoTree())
		// filter by product codes
		filtered_props = filterPropertiesByProductCodes(filtered_props, query.SearchParametersAccommodation.GetProductCodes())
		// filter by supplier codes
		filtered_props = filterPropertiesBySupplierCodes(filtered_props, query.SearchParametersAccommodation.GetSupplierCodes())

		// loop filtered properties and check if they are already in available_properties
		for _, prop := range filtered_props {
			// Check if property already exists in available_properties
			exists := false
			for _, existingProp := range available_properties {
				if existingProp.Property.SupplierCode.SupplierCode == prop.Property.SupplierCode.SupplierCode {
					exists = true
					break
				}
			}
			if !exists {
				available_properties = append(available_properties, prop)
			}
		}

		var price = 500

		// generate search result
		for _, prop := range available_properties {

			// units requested
			units_requested := query.UnitCount

			// empty units array
			units := make([]*accommodationv2.Unit, 0)

			// loop all rooms
			for _, room := range prop.Rooms {

				units = append(units, &accommodationv2.Unit{
					Type:             0,
					SupplierRoomCode: room.SupplierCode,
					SupplierRoomName: room.SupplierName,
					OriginalRoomName: room.OriginalName,
					TravelPeriod: &typesv1.TravelPeriod{
						StartDate: &typesv1.Date{
							Year:  query.TravelPeriod.GetStartDate().GetYear(),
							Month: query.TravelPeriod.GetStartDate().GetMonth(),
							Day:   query.TravelPeriod.GetStartDate().GetDay(),
						},
						EndDate: &typesv1.Date{
							Year:  query.TravelPeriod.GetEndDate().GetYear(),
							Month: query.TravelPeriod.GetEndDate().GetMonth(),
							Day:   query.TravelPeriod.GetEndDate().GetDay(),
						},
					},
					TravellerIds: getTravellerIds(query.Travellers),
					Beds:         room.Beds,
					PriceDetail: &typesv2.PriceDetail{
						Price: &typesv2.Price{
							Value: fmt.Sprintf("%d", price),
							Currency: &typesv2.Currency{
								Currency: req.SearchParametersGeneric.Currency.Currency,
							},
						},
					},
					Services:       []*typesv2.ServiceFact{},
					MealPlanCode:   &typesv1.MealPlan{},
					RatePlan:       &typesv1.RatePlan{},
					RateRule:       &typesv1.RateRule{},
					CancelPolicies: []*typesv2.CancelPolicy{},
					RemainingUnits: 0,
					PropertyCode:   &typesv2.ProductCode{},
					SupplierCode:   prop.Property.SupplierCode,
					Remarks:        "",
				})

				price += 250

				if units_requested == int32(len(units)) {
					break
				}
			}

			// check how many units are requested
			if units_requested == int32(len(units)) {
				searchResults = append(searchResults, &accommodationv2.AccommodationSearchResult{
					ResultId: int32(len(searchResults) + 1),
					QueryId:  query.QueryId,
					TotalPriceDetail: &typesv2.PriceDetail{
						Price: &typesv2.Price{
							Value: fmt.Sprintf("%d", price),
							Currency: &typesv2.Currency{
								Currency: req.SearchParametersGeneric.Currency.Currency,
							},
						},
					},
					Units: units,
				})
			}
		}
	}

	if len(searchResults) == 0 {
		return &accommodationv2.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{
					{Message: fmt.Sprintf("No results found for search %v", req.Queries)},
				},
			},
		}, nil
	}

	cache := cache.NewSearchCache()
	// Store in cache after search

	searchId := uuid.New().String()

	cache.SetV2(searchId, searchResults)

	response := &accommodationv2.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv2.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchId},
		},
		Results: searchResults,
		Travellers: []*typesv2.BasicTraveller{{
			Type:        typesv2.TravellerType(typesv1.TravelType_TRAVEL_TYPE_LEISURE),
			Birthdate:   &typesv1.Date{},
			Nationality: typesv2.Country_COUNTRY_DE,
		}},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

// FilterPropertiesByGeoTreeLocation filters properties based on city or resort
func filterPropertiesByGeoTreeLocation(properties []*accommodationv2.PropertyExtendedInfo, geoTreeLocation *typesv2.GeoTree) []*accommodationv2.PropertyExtendedInfo {
	if geoTreeLocation == nil || geoTreeLocation.CityOrResort == "" || geoTreeLocation.Region == "" {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		var address = prop.Property.ContactInfo.Address[0]
		if address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort && address.GeoTree.Country == geoTreeLocation.Country && address.GeoTree.Region == geoTreeLocation.Region {
			filtered = append(filtered, prop)
		}
	}

	return filtered
}

// getTravellerIds extracts traveller IDs from []*typesv1.BasicTraveller
func getTravellerIds(travellers []*typesv2.BasicTraveller) []int32 {
	var ids []int32
	for _, traveller := range travellers {
		ids = append(ids, traveller.TravellerId)
	}
	return ids
}

// filterPropertiesByProductCodes filters properties based on product codes
func filterPropertiesByProductCodes(properties []*accommodationv2.PropertyExtendedInfo, productCodes []*typesv2.ProductCode) []*accommodationv2.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		for _, code := range productCodes {
			if prop.Property.ProductCodes[0].Code == code.Code {
				filtered = append(filtered, prop)
				break
			}
		}
	}
	return filtered
}

// filterPropertiesBySupplierCodes filters properties based on supplier codes
func filterPropertiesBySupplierCodes(properties []*accommodationv2.PropertyExtendedInfo, supplierCodes []*typesv2.SupplierProductCode) []*accommodationv2.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		for _, code := range supplierCodes {
			if prop.Property.SupplierCode.SupplierCode == code.SupplierCode {
				filtered = append(filtered, prop)
				break
			}
		}
	}
	return filtered
}
