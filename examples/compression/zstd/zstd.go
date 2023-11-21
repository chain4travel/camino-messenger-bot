/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package main

import (
	"os"

	"github.com/klauspost/compress/zstd"
)

const (
	FixedBlockSize = 48 << 10 // according to element limit
)

var encoder, _ = zstd.NewWriter(nil, zstd.WithWindowSize(FixedBlockSize))
var decoder, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))

func main() {
	wd, _ := os.Getwd()
	bytes, err := os.ReadFile(wd + "/examples/compression/test.txt")
	if err != nil {
		panic(err)
	}

	compressedBytes := Compress(bytes)
	os.WriteFile("ctest", compressedBytes, 0644)

	decompressedBytes, err := Decompress(compressedBytes)
	if err != nil {
		panic(err)
	}
	os.WriteFile("dtest.txt", decompressedBytes, 0644)
}

// Compress a buffer.
// If you have a destination buffer, the allocation in the call can also be eliminated.
func Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func Decompress(src []byte) ([]byte, error) {
	return decoder.DecodeAll(src, nil)
}
