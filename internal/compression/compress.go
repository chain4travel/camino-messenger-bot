// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package compression

import (
	"github.com/klauspost/compress/zstd"
)

var encoder, _ = zstd.NewWriter(nil)

// Compressor interface defines basic compression functionality
type Compressor[T any, R any] interface {
	Compress(data T) (R, error)
}

// CompressBytes takes a byte array as input and returns the compressed data as a byte array
func CompressBytes(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}
