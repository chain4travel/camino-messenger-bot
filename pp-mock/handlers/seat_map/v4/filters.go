// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterSeatMapByID(
	seatMaps []*typesv4.SeatMap,
	mapID *typesv4.SeatMapID,
) *typesv4.SeatMap {
	for _, seatMap := range seatMaps {
		if proto.Equal(seatMap.Id, mapID) {
			return common.CloneProto(seatMap)
		}
	}
	return nil
}

func filterSeatMapAvailabilityByID(
	seatMaps []*typesv4.SeatMapInventory,
	mapID string,
) *typesv4.SeatMapInventory {
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

	for i, s := range section.GetSubsections().GetSections() {
		missingChildLocalization := false
		section.GetSubsections().Sections[i], missingChildLocalization = filterSeatMapSectionLocalization(s, langSet)
		missingLocalization = missingLocalization || missingChildLocalization
	}

	return section, missingLocalization
}
