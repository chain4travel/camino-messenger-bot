// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package localization

import (
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

func FilterDescriptionsV4(descriptions []*typesv4.LocalizedDescriptionSet, languages []typesv1.Language) []*typesv4.LocalizedDescriptionSet {
	filteredDescriptions := []*typesv4.LocalizedDescriptionSet{}

	for _, descSet := range descriptions {
		for _, reqLang := range languages {
			if descSet.Language == reqLang {
				filteredDescriptions = append(filteredDescriptions, descSet)
				break
			}
		}
	}

	return filteredDescriptions
}
