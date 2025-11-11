// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cancellation

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	cancellationReasonVersion = 1
	withdrawReasonVersion     = 1
	rejectReasonVersion       = 1
	counterReasonVersion      = 1
)

// Safely converts an int32 to uint16, returning an error if out of range.
func uint16FromProtoEnumNumber(value protoreflect.EnumNumber) (uint16, error) {
	if value < 0 || value > math.MaxUint16 {
		return 0, fmt.Errorf("value out of range for uint16: %d", value)
	}
	// TODO @VjeraTurk check if false positives are removed in newer versions
	// https://github.com/securego/gosec/issues/1212
	// https://github.com/securego/gosec/pull/1194

	// Otherwise lint.sh fails with G115: integer overflow conversion int32 -> uint16 (gosec)
	// nolint:gosec
	return uint16(value), nil
}
