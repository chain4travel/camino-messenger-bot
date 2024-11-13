package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/cache"
	helpers "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/data/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationSearchServiceServer = (*AccommodationSearchV1Server)(nil)

type AccommodationSearchV1Server struct{}

func (*AccommodationSearchV1Server) AccommodationSearch(ctx context.Context, req *accommodationv1.AccommodationSearchRequest) (*accommodationv1.AccommodationSearchResponse, error) {
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
	properties := helpers.LoadPropertiesMockData()

	// log
	fmt.Printf("properties: %+v\n", properties)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &accommodationv1.AccommodationSearchResponse{
			Header: nil,
		}, nil
	}

	var searchResults []*accommodationv1.AccommodationSearchResult
	var available_properties []*accommodationv1.PropertyExtendedInfo
	// loop request queries
	for _, query := range req.Queries {
		props := make([]*accommodationv1.PropertyExtendedInfo, len(properties))
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
			units := make([]*accommodationv1.Unit, 0)

			// loop all rooms
			for _, room := range prop.Rooms {

				units = append(units, &accommodationv1.Unit{
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
					PriceDetail: &typesv1.PriceDetail{
						Price: &typesv1.Price{
							Value: fmt.Sprintf("%d", price),
							Currency: &typesv1.Currency{
								Currency: req.SearchParametersGeneric.Currency.Currency,
							},
						},
					},
					Services:       []*typesv1.ServiceFact{},
					MealPlanCode:   &typesv1.MealPlan{},
					RatePlan:       &typesv1.RatePlan{},
					RateRule:       &typesv1.RateRule{},
					CancelPolicies: []*typesv1.CancelPolicy{},
					RemainingUnits: 0,
					PropertyCode:   &typesv1.ProductCode{},
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
				searchResults = append(searchResults, &accommodationv1.AccommodationSearchResult{
					ResultId: int32(len(searchResults) + 1),
					QueryId:  query.QueryId,
					TotalPriceDetail: &typesv1.PriceDetail{
						Price: &typesv1.Price{
							Value: fmt.Sprintf("%d", price),
							Currency: &typesv1.Currency{
								Currency: req.SearchParametersGeneric.Currency.Currency,
							},
						},
					},
					Units: units,
				})
			}
		}
	}

	// generate a random string of 8 numbers
	cache := cache.NewSearchCache()

	searchId := uuid.New().String()

	// Store in cache after search
	cache.SetV1(searchId, searchResults)

	response := &accommodationv1.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{},
		Metadata: &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchId},
		},
		Results: searchResults,
		Travellers: []*typesv1.BasicTraveller{{
			Type:        typesv1.TravellerType(typesv1.TravelType_TRAVEL_TYPE_LEISURE),
			Birthdate:   &typesv1.Date{},
			Nationality: typesv1.Country_COUNTRY_DE,
		}},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

// FilterPropertiesByGeoTreeLocation filters properties based on city or resort
func filterPropertiesByGeoTreeLocation(properties []*accommodationv1.PropertyExtendedInfo, geoTreeLocation *typesv1.GeoTree) []*accommodationv1.PropertyExtendedInfo {
	if geoTreeLocation == nil || geoTreeLocation.CityOrResort == "" || geoTreeLocation.Region == "" {
		return properties
	}

	filtered := make([]*accommodationv1.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		var address = prop.Property.ContactInfo.Address[0]
		if address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort && address.GeoTree.Country == geoTreeLocation.Country && address.GeoTree.Region == geoTreeLocation.Region {
			filtered = append(filtered, prop)
		}
	}

	return filtered
}

// getTravellerIds extracts traveller IDs from []*typesv1.BasicTraveller
func getTravellerIds(travellers []*typesv1.BasicTraveller) []int32 {
	var ids []int32
	for _, traveller := range travellers {
		ids = append(ids, traveller.TravellerId)
	}
	return ids
}

// filterPropertiesByProductCodes filters properties based on product codes
func filterPropertiesByProductCodes(properties []*accommodationv1.PropertyExtendedInfo, productCodes []*typesv1.ProductCode) []*accommodationv1.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv1.PropertyExtendedInfo, 0)
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
func filterPropertiesBySupplierCodes(properties []*accommodationv1.PropertyExtendedInfo, supplierCodes []*typesv1.SupplierProductCode) []*accommodationv1.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv1.PropertyExtendedInfo, 0)
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
