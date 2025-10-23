// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"time"

	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

// Filters properties based on product codes
func filterExtendedPropertiesByProductCodes(
	properties []*accommodationv4.PropertyExtendedInfo,
	productCodes []*typesv4.ProductCode,
) []*accommodationv4.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv4.PropertyExtendedInfo{}
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
	properties []*accommodationv4.PropertyExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*accommodationv4.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv4.PropertyExtendedInfo{}
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
	properties []*accommodationv4.PropertyExtendedInfo,
	geoTreeLocation *typesv4.GeoTree,
) []*accommodationv4.PropertyExtendedInfo {
	if geoTreeLocation == nil ||
		(geoTreeLocation.CityOrResort == "" &&
			geoTreeLocation.Region == "" &&
			geoTreeLocation.Country == typesv2.Country_COUNTRY_UNSPECIFIED) {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv4.PropertyExtendedInfo{}
	for _, prop := range properties {
		// Mock simplification: mock data only has one address
		address := prop.Property.ContactInfo.Addresses[0]
		if address.GeoTree != nil &&
			(geoTreeLocation.CityOrResort == "" || address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort) &&
			(geoTreeLocation.Country == typesv2.Country_COUNTRY_UNSPECIFIED || address.GeoTree.Country == geoTreeLocation.Country) &&
			(geoTreeLocation.Region == "" || address.GeoTree.Region == geoTreeLocation.Region) {
			filtered = append(filtered, common.CloneProto(prop))
		}
	}

	return filtered
}

// Filters properties based on languages. Will not modify original slice or its elements.
func filterExtendedPropertiesByLanguage(
	properties []*accommodationv4.PropertyExtendedInfo,
	languages []typesv1.Language,
) []*accommodationv4.PropertyExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(properties)
	}

	filtered := []*accommodationv4.PropertyExtendedInfo{}
	for _, property := range properties {
		filteredDescriptions := []*typesv4.LocalizedDescriptionSet{}

		for _, descSet := range property.LocalizedDescriptions {
			for _, reqLang := range languages {
				if descSet.Language == reqLang {
					filteredDescriptions = append(filteredDescriptions, common.CloneProto(descSet))
					break
				}
			}
		}

		if len(filteredDescriptions) > 0 {
			clonedProperty := common.CloneProto(property)
			clonedProperty.LocalizedDescriptions = filteredDescriptions
			filtered = append(filtered, clonedProperty)
		}
	}
	return filtered
}

// Filters properties based on supplier codes. Will not modify original slice or its elements.
func filterPropertiesBySupplierCodes(
	properties []*accommodationv4.PropertyExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*accommodationv4.Property {
	filtered := []*accommodationv4.Property{}

	if len(supplierCodes) == 0 {
		for _, property := range properties {
			filtered = append(filtered, common.CloneProto(property.Property))
		}
		return filtered
	}

	for _, property := range properties {
		for _, code := range supplierCodes {
			if proto.Equal(property.Property.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(property.Property))
				break
			}
		}
	}

	return filtered
}

// Returns properties that have been modified not before [lastModified].
func filterShortItemsByModifierAfter(
	properties []*accommodationv4.PropertyExtendedInfo,
	modifiedAfter time.Time,
) []*accommodationv4.PropertyShortListItem {
	filtered := []*accommodationv4.PropertyShortListItem{}
	for _, property := range properties {
		if !property.Property.LastModified.AsTime().Before(modifiedAfter) {
			filtered = append(filtered, &accommodationv4.PropertyShortListItem{
				SupplierCode: property.Property.SupplierCode,
				Status:       property.Property.Status,
			})
		}
	}
	return filtered
}
