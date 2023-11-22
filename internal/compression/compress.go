/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package compression

import (
	"encoding/base64"

	"github.com/klauspost/compress/zstd"
)

const (
	MaxChunkSize = 48 << 10 // according to element configured safe max limit
)

var encoder, _ = zstd.NewWriter(nil) // TODO: evalaute the need of using zstd.WithEncoderConcurrency

func Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}
func CompressAndEncode(src []byte) string {
	bytes := Compress(src)
	return base64.StdEncoding.EncodeToString(bytes)
}
