package v3

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

func filterSeatMapLanguage(
	seatMap *typesv4.SeatMap,
	languages []typesv1.Language,
) (*typesv4.SeatMap, []*typesv4.Alert) {
	filteredMap := common.CloneProto(seatMap)
	if len(languages) == 0 {
		return filteredMap, nil
	}

	// TODO@ if we filter out all langs, than entry is without localized strings, its ok; but if this happens, we add warning to alerts

	var alerts []*typesv1.Alert
	var descriptions []*typesv4.LocalizedDescriptionSet
	for _, desc := range seatMap.Descriptions {
		for _, lang := range languages {
			if desc.Language == lang {
				descriptions = append(descriptions, desc)
				break
			}
		}
	}
	filteredMap.Descriptions = descriptions

	for i, section := range filteredMap.Sections {
		filteredMap.Sections[i] = filterSeatMapSectionLanguage(section, languages)
	}

	return filteredMap, alerts
}

func filterSeatMapSectionLanguage(
	section *typesv4.Section,
	languages []typesv1.Language,
) (*typesv4.Section, []*typesv4.Alert) {
	filteredSection := common.CloneProto(section)
	if len(languages) == 0 {
		return filteredSection, nil
	}

	for _, section := range filteredSection.Sections {
		var descriptions []*typesv4.LocalizedDescriptionSet
		for _, desc := range section.Descriptions {
			for _, lang := range languages {
				if desc.Language == lang {
					descriptions = append(descriptions, desc)
					break
				}
			}
		}
		section.Descriptions = descriptions
	}

	return filteredSection
}
