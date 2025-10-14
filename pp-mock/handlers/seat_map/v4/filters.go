// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
)

func filterSeatMapByID(
	seatMaps []*typesv4.SeatMap,
	mapID string,
) *typesv4.SeatMap {
	for _, seatMap := range seatMaps {
		if seatMap.Id.Id == mapID {
			return common.CloneProto(seatMap)
		}
	}
	return nil
}

func filterSeatMapLocalization(
	seatMap *typesv4.SeatMap,
	languages []typesv1.Language,
) (*typesv4.SeatMap, bool) {
	langSet := make(map[typesv1.Language]struct{})
	for _, lang := range languages {
		langSet[lang] = struct{}{}
	}

	filteredMap := common.CloneProto(seatMap)

	filteredMap.Attributes = filterSeatAttributesLocalization(filteredMap.Attributes, langSet)
	missingLocalization := false

	for i, section := range filteredMap.Sections {
		missingChildLocalization := false
		filteredMap.Sections[i], missingChildLocalization = filterSeatMapSectionLocalization(section, langSet)
		missingLocalization = missingLocalization || missingChildLocalization
	}

	return filteredMap, missingLocalization
}

func filterSeatMapSectionLocalization(
	section *typesv4.Section,
	langSet map[typesv1.Language]struct{},
) (*typesv4.Section, bool) {
	if len(langSet) == 0 {
		return section, false
	}

	originalSectionNames := section.Names
	section.Names = nil
	for _, name := range originalSectionNames {
		if _, ok := langSet[name.Language]; ok {
			section.Names = append(section.Names, name)
		}
	}

	missingLocalization := len(originalSectionNames) != 0 && len(section.Names) == 0

	section.Attributes = filterSeatAttributesLocalization(section.Attributes, langSet)

	if seatList, ok := section.SeatInfo.(*typesv4.Section_SeatList); ok {
		for _, seat := range seatList.SeatList.Seats {
			seat.Attributes = filterSeatAttributesLocalization(seat.Attributes, langSet)
		}
	}

	for i, s := range section.Sections {
		missingChildLocalization := false
		section.Sections[i], missingChildLocalization = filterSeatMapSectionLocalization(s, langSet)
		missingLocalization = missingLocalization || missingChildLocalization
	}

	return section, missingLocalization
}

func filterSeatAttributesLocalization(
	attributes *typesv4.SeatAttributes,
	langSet map[typesv1.Language]struct{},
) *typesv4.SeatAttributes {
	if len(langSet) == 0 || attributes == nil {
		return attributes
	}

	originalDescriptions := attributes.Descriptions
	attributes.Descriptions = nil
	for _, desc := range originalDescriptions {
		if _, ok := langSet[desc.Language]; ok {
			attributes.Descriptions = append(attributes.Descriptions, desc)
		}
	}

	originalFeatures := attributes.Features
	attributes.Features = nil
	for _, feature := range originalFeatures {
		if _, ok := langSet[feature.Language]; ok {
			attributes.Features = append(attributes.Features, feature)
		}
	}

	originalRestrictions := attributes.Restrictions
	attributes.Restrictions = nil
	for _, restriction := range originalRestrictions {
		if _, ok := langSet[restriction.Language]; ok {
			attributes.Restrictions = append(attributes.Restrictions, restriction)
		}
	}

	return attributes
}
