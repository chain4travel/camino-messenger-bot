package v3

import (
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
)

func filterSeatMapByID(
	seatMaps []*typesv3.SeatMap,
	mapID string,
) *typesv3.SeatMap {
	for _, seatMap := range seatMaps {
		if seatMap.Id == mapID {
			return common.CloneProto(seatMap)
		}
	}
	return nil
}

func filterSeatMapLanguage(
	seatMap *typesv3.SeatMap,
	languages []typesv1.Language,
) (*typesv3.SeatMap, []*typesv1.Alert) {
	filteredMap := common.CloneProto(seatMap)
	if len(languages) == 0 {
		return filteredMap, nil
	}

	// TODO@ if we filter out all langs, than entry is without localized strings, its ok; but if this happens, we add warning to alerts

	var alerts []*typesv1.Alert
	var localizedDescriptions []*typesv1.LocalizedDescriptionSet
	for _, desc := range seatMap.LocalizedDescriptions {
		for _, lang := range languages {
			if desc.Language == lang {
				localizedDescriptions = append(localizedDescriptions, desc)
				break
			}
		}
	}
	filteredMap.LocalizedDescriptions = localizedDescriptions

	for i, section := range filteredMap.Sections {
		filteredMap.Sections[i] = filterSeatMapSectionLanguage(section, languages)
	}

	return filteredMap, alerts
}

func filterSeatMapSectionLanguage(
	section *typesv3.Section,
	languages []typesv1.Language,
) (*typesv3.Section, []*typesv1.Alert) {
	filteredSection := common.CloneProto(section)
	if len(languages) == 0 {
		return filteredSection, nil
	}

	for _, section := range filteredSection.Sections {
		var localizedSectionDescriptions []*typesv1.LocalizedDescriptionSet
		for _, desc := range section.LocalizedDescriptions {
			for _, lang := range languages {
				if desc.Language == lang {
					localizedSectionDescriptions = append(localizedSectionDescriptions, desc)
					break
				}
			}
		}
		section.LocalizedDescriptions = localizedSectionDescriptions
	}

	return filteredSection
}
