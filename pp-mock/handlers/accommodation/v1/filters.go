// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"time"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	"google.golang.org/protobuf/proto"
)

// Filters properties based on product codes
func filterExtendedPropertiesByProductCodes(
	properties []*accommodationv1.PropertyExtendedInfo,
	productCodes []*typesv1.ProductCode,
) []*accommodationv1.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv1.PropertyExtendedInfo{}
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
	properties []*accommodationv1.PropertyExtendedInfo,
	supplierCodes []*typesv1.SupplierProductCode,
) []*accommodationv1.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv1.PropertyExtendedInfo{}
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
	properties []*accommodationv1.PropertyExtendedInfo,
	geoTreeLocation *typesv1.GeoTree,
) []*accommodationv1.PropertyExtendedInfo {
	if geoTreeLocation == nil ||
		(geoTreeLocation.CityOrResort == "" &&
			geoTreeLocation.Region == "" &&
			geoTreeLocation.Country == typesv1.Country_COUNTRY_UNSPECIFIED) {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv1.PropertyExtendedInfo{}
	for _, prop := range properties {
		// Mock simplification: mock data only has one address
		address := prop.Property.ContactInfo.Address[0]
		if address.GeoTree != nil &&
			(geoTreeLocation.CityOrResort == "" || address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort) &&
			(geoTreeLocation.Country == typesv1.Country_COUNTRY_UNSPECIFIED || address.GeoTree.Country == geoTreeLocation.Country) &&
			(geoTreeLocation.Region == "" || address.GeoTree.Region == geoTreeLocation.Region) {
			filtered = append(filtered, common.CloneProto(prop))
		}
	}

	return filtered
}

// Filters properties based on supplier codes. Will not modify original slice or its elements.
func filterExtendedPropertiesByLanguage(
	properties []*accommodationv1.PropertyExtendedInfo,
	languages []typesv1.Language,
) []*accommodationv1.PropertyExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv1.PropertyExtendedInfo{}
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

		if (len(filteredDescriptions) > 0 || len(filteredRoomDescriptions) > 0) &&
			!containsPropertyWithSupplierCode(filtered, property) {
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
	properties []*accommodationv1.PropertyExtendedInfo,
	lastModified time.Time,
) []*accommodationv1.Property {
	filtered := []*accommodationv1.Property{}
	for _, property := range properties {
		if !property.Property.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(property.Property))
		}
	}
	return filtered
}

// Returns true if [properties] contains property with matching supplier code.
func containsPropertyWithSupplierCode(
	properties []*accommodationv1.PropertyExtendedInfo,
	property *accommodationv1.PropertyExtendedInfo,
) bool {
	for _, p := range properties {
		if proto.Equal(p.Property.SupplierCode, property.Property.SupplierCode) {
			return true
		}
	}
	return false
}
