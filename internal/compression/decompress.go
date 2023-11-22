/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package compression

import (
	"github.com/klauspost/compress/zstd"
)

var decoder, _ = zstd.NewReader(nil)

func Decompress(src []byte) ([]byte, error) {
	return decoder.DecodeAll(src, nil)
}
