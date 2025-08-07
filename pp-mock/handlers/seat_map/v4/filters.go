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
		if seatMap.Id == mapID {
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
	missingLocalization := false

	originalDescriptionsLen := len(filteredMap.Descriptions)
	filteredMap.Descriptions = nil
	for _, desc := range seatMap.Descriptions {
		if _, ok := langSet[desc.Language]; ok {
			filteredMap.Descriptions = append(filteredMap.Descriptions, desc)
		}
	}

	if originalDescriptionsLen != 0 && len(filteredMap.Descriptions) == 0 {
		missingLocalization = true
	}

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

	missingLocalization := false

	originalSectionNames := section.Names
	section.Names = nil
	for _, name := range originalSectionNames {
		if _, ok := langSet[name.Language]; ok {
			section.Names = append(section.Names, name)
		}
	}

	originalSectionDescriptions := section.Descriptions
	section.Descriptions = nil
	for _, desc := range originalSectionDescriptions {
		if _, ok := langSet[desc.Language]; ok {
			section.Descriptions = append(section.Descriptions, desc)
		}
	}

	if seatList, ok := section.SeatInfo.(*typesv4.Section_SeatList); ok {
		for _, seat := range seatList.SeatList.Seats {
			originalSeatFeatures := seat.Features
			seat.Features = nil
			for _, feature := range originalSeatFeatures {
				if _, ok := langSet[feature.Language]; ok {
					seat.Features = append(seat.Features, feature)
				}
			}

			originalSeatRestrictions := seat.Restrictions
			seat.Restrictions = nil
			for _, restriction := range originalSeatRestrictions {
				if _, ok := langSet[restriction.Language]; ok {
					seat.Restrictions = append(seat.Restrictions, restriction)
				}
			}

			if len(originalSeatFeatures) != 0 && len(seat.Features) == 0 ||
				len(originalSeatRestrictions) != 0 && len(seat.Restrictions) == 0 {
				missingLocalization = true
			}
		}
	}

	if len(originalSectionNames) != 0 && len(section.Names) == 0 ||
		len(originalSectionDescriptions) != 0 && len(section.Descriptions) == 0 {
		missingLocalization = true
	}

	for i, s := range section.Sections {
		missingChildLocalization := false
		section.Sections[i], missingChildLocalization = filterSeatMapSectionLocalization(s, langSet)
		missingLocalization = missingLocalization || missingChildLocalization
	}

	return section, missingLocalization
}
