// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"time"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

// Filters properties based on product codes
func filterExtendedPropertiesByProductCodes(
	properties []*accommodationv2.PropertyExtendedInfo,
	productCodes []*typesv2.ProductCode,
) []*accommodationv2.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv2.PropertyExtendedInfo{}
	for _, prop := range properties {
	productCodesLoop:
		for _, code := range productCodes {
			for _, propCode := range prop.Property.ProductCodes {
				if proto.Equal(propCode, code) {
					filtered = append(filtered, common.CloneProto(prop))
					break productCodesLoop
				}
			}
		}
	}
	return filtered
}

// Filters properties based on supplier codes. Will not modify original slice or its elements.
func filterExtendedPropertiesBySupplierCodes(
	properties []*accommodationv2.PropertyExtendedInfo,
	supplierCodes []*typesv2.SupplierProductCode,
) []*accommodationv2.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv2.PropertyExtendedInfo{}
	for _, prop := range properties {
		for _, code := range supplierCodes {
			if proto.Equal(prop.Property.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(prop))
				break
			}
		}
	}
	return filtered
}

// Filters properties based on city or resort
func filterExtendedPropertiesByGeoTreeLocation(
	properties []*accommodationv2.PropertyExtendedInfo,
	geoTreeLocation *typesv2.GeoTree,
) []*accommodationv2.PropertyExtendedInfo {
	if geoTreeLocation == nil ||
		(geoTreeLocation.CityOrResort == "" &&
			geoTreeLocation.Region == "" &&
			geoTreeLocation.Country == typesv2.Country_COUNTRY_UNSPECIFIED) {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv2.PropertyExtendedInfo{}
	for _, prop := range properties {
		// Mock simplification: mock data only has one address
		address := prop.Property.ContactInfo.Address[0]
		if address.GeoTree != nil &&
			(geoTreeLocation.CityOrResort == "" || address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort) &&
			(geoTreeLocation.Country == typesv2.Country_COUNTRY_UNSPECIFIED || address.GeoTree.Country == geoTreeLocation.Country) &&
			(geoTreeLocation.Region == "" || address.GeoTree.Region == geoTreeLocation.Region) {
			filtered = append(filtered, common.CloneProto(prop))
		}
	}

	return filtered
}

// Filters properties based on supplier codes. Will not modify original slice or its elements.
func filterExtendedPropertiesByLanguage(
	properties []*accommodationv2.PropertyExtendedInfo,
	languages []typesv1.Language,
) []*accommodationv2.PropertyExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv2.PropertyExtendedInfo{}
	for _, property := range properties {
		filteredDescriptions := []*typesv1.LocalizedDescriptionSet{}
		filteredRoomDescriptions := []*typesv1.LocalizedDescriptionSet{}

		for _, descSet := range property.LocalizedDescriptions {
			for _, reqLang := range languages {
				if descSet.Language == reqLang {
					filteredDescriptions = append(filteredDescriptions, descSet)
					break
				}
			}
		}

		for _, roomDescSet := range property.LocalizedRoomDescriptions {
			for _, reqLang := range languages {
				if roomDescSet.Language == reqLang {
					filteredRoomDescriptions = append(filteredRoomDescriptions, roomDescSet)
					break
				}
			}
		}

		if len(filteredDescriptions) > 0 || len(filteredRoomDescriptions) > 0 {
			clonedProperty := common.CloneProto(property)
			clonedProperty.LocalizedDescriptions = filteredDescriptions
			clonedProperty.LocalizedRoomDescriptions = filteredRoomDescriptions
			filtered = append(filtered, clonedProperty)
		}
	}
	return filtered
}

// Returns properties that have been modified not before [lastModified].
func filterPropertiesByLastModified(
	properties []*accommodationv2.PropertyExtendedInfo,
	lastModified time.Time,
) []*accommodationv2.Property {
	filtered := []*accommodationv2.Property{}
	for _, property := range properties {
		if !property.Property.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(property.Property))
		}
	}
	return filtered
}
