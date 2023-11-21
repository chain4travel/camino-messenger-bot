/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/pierrec/lz4"
)

const chunkSize = 48 << 10 // 48KB according to element "safe" limit

func compress(in []byte) ([]byte, error) {
	r := bytes.NewReader(in)
	w := &bytes.Buffer{}
	zw := lz4.NewWriter(w)
	_, err := io.Copy(zw, r)
	if err != nil {
		return nil, err
	}
	// Closing is *very* important
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func compressAndSplit(inputFilePath, outputDirectory string) (uint, error) {
	bytes, err := os.ReadFile(inputFilePath)
	if err != nil {
		return 0, err
	}
	compressedData, err := compress(bytes)

	// Perform chunking on the compressed content and write chunks to separate files
	for i := 0; i < len(compressedData); i += chunkSize {
		end := i + chunkSize
		if end > len(compressedData) {
			end = len(compressedData)
		}

		chunk := compressedData[i:end]

		// Create a new file for writing compressed chunk
		compressedChunkFilePath := fmt.Sprintf("%s_%s.lz4", outputDirectory, strconv.Itoa(i/chunkSize))
		compressedChunkFile, err := os.Create(compressedChunkFilePath)
		if err != nil {
			return 0, fmt.Errorf("error creating compressed file: %v", err)
		}

		// Write the compressed chunk
		_, err = compressedChunkFile.Write(chunk)
		if err != nil {
			return 0, fmt.Errorf("error writing compressed data to file: %v", err)
		}

		// Close the compressed chunk file
		err = compressedChunkFile.Close()
		if err != nil {
			return 0, fmt.Errorf("error closing compressed file: %v", err)
		}

		log.Printf("Compression successful for chunk %d", i/chunkSize)
	}

	return uint(math.Ceil(float64(len(compressedData)) / chunkSize)), nil
}

func combineAndDecompress(inputPrefix, outputFilePath string, numberOfChunks uint) error {
	var compressedChunks []byte
	for i := uint(0); i < numberOfChunks; i++ {
		inputFilePath := fmt.Sprintf("%s_%d.lz4", inputPrefix, i)
		compressedChunk, err := os.ReadFile(inputFilePath)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		compressedChunks = append(compressedChunks, compressedChunk...)
	}

	// Create a decompressor
	r := lz4.NewReader(bytes.NewReader(compressedChunks))

	// Decompress data
	decompressedData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("Error decompressing data: %v", err)
	}

	// Write decompressed data to a new file
	err = os.WriteFile(outputFilePath, decompressedData, 0644)
	if err != nil {
		log.Fatalf("Error writing decompressed data to file: %v", err)
	}

	return nil
}

func main() {
	wd, _ := os.Getwd()

	inputFilePath := wd + "/examples/compression/test.pdf"
	outputPrefix := wd + "/my_large_file_split"
	outputFilePath := wd + "/my_large_file_combined.pdf"

	// Compress and split the file
	numberOfChunks, err := compressAndSplit(inputFilePath, outputPrefix)
	if err != nil {
		fmt.Println("Error compressing and splitting:", err)
		return
	}

	// Combine and decompress the chunks
	if err := combineAndDecompress(outputPrefix, outputFilePath, numberOfChunks); err != nil {
		fmt.Println("Error combining and decompressing:", err)
		return
	}

	fmt.Println("Compression and decompression with chunking completed.")
}
