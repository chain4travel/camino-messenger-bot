/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package compression

import (
	"github.com/klauspost/compress/zstd"
)

const (

	// MaxChunkSize a moderate/safe max chunk size is 48KB. This is because the maximum size of a matrix event is 64KB.
	// Megolm encryption adds an extra 33% overhead to the encrypted content due to base64 encryption. This means that
	// the maximum size of pre-encrypted chunk should be 48KB / 1.33 ~= 36KB. We round down to 35KB to be safe.
	MaxChunkSize = 35 << 10 // max pre-encrypted chunk size is 35KB
)

var encoder, _ = zstd.NewWriter(nil)

func Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}
