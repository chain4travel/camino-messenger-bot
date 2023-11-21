/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
)

func main2() {
	var inputFile string
	var outputPrefix string
	var chunkSize int

	wd, _ := os.Getwd()
	inputFilePath := wd + "/examples/compression/test.pdf"

	flag.StringVar(&inputFile, "input", inputFilePath, "Input file to be split")
	flag.StringVar(&outputPrefix, "output-prefix", "output", "Output file prefix")
	flag.IntVar(&chunkSize, "chunk-size", 1000, "Size of each chunk in lines")
	flag.Parse()

	if inputFile == "" {
		fmt.Println("Please provide an input file using the -input flag")
		os.Exit(1)
	}

	// Open the input file
	input, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening input file: %s\n", err)
		os.Exit(1)
	}
	defer input.Close()

	// Initialize variables
	buf := make([]byte, 1024)
	var chunkCount int
	var currentChunkSize int
	hasher := sha256.New()

	// Create and open the first output file
	outputFile, err := createOutFile(outputPrefix, chunkCount)
	if err != nil {
		fmt.Printf("Error creating output file: %s\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// Iterate over the input file
	for {
		// Read a chunk of data from the input file
		n, err := input.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Printf("Error reading input file: %s\n", err)
			os.Exit(1)
		}

		// Update the hash with the current chunk
		hasher.Write(buf[:n])

		// Write the chunk to the current output file
		_, err = outputFile.Write(buf[:n])
		if err != nil {
			fmt.Printf("Error writing to output file: %s\n", err)
			os.Exit(1)
		}

		currentChunkSize += n

		fmt.Printf("Chunk %d: %d bytes\n", chunkCount, n)
		// Check if the current chunk size exceeds the specified chunk size
		if currentChunkSize >= chunkSize {
			// Close the current output file
			outputFile.Close()

			// Calculate the checksum for the current file
			checksum := fmt.Sprintf("%x", hasher.Sum(nil))

			// Append checksum and reference to the next split file to the current file
			appendReferenceToCurrentFile(outputPrefix, chunkCount, checksum, chunkCount+1)

			// Reset the hash and create and open the next output file
			hasher = sha256.New()
			outputFile, err = createOutFile(outputPrefix, chunkCount+1)
			if err != nil {
				fmt.Printf("Error creating output file: %s\n", err)
				os.Exit(1)
			}

			// Reset the current chunk size
			currentChunkSize = 0
			fmt.Printf("currentChunkSize %d: %d bytes\n", currentChunkSize, n)
		}

		// Break the loop if we've reached the end of the input file
		if err == io.EOF {
			break
		}
	}

	// Close the last output file
	outputFile.Close()

	fmt.Println("Split operation completed successfully.")
}

// createOutFile creates a new output file with the specified prefix and chunk count.
func createOutFile(prefix string, count int) (*os.File, error) {
	filename := fmt.Sprintf("%s_%d", prefix, count)
	return os.Create(filename)
}

// appendReferenceToCurrentFile appends the checksum and reference to the next split file to the current file.
func appendReferenceToCurrentFile(prefix string, count int, checksum string, nextCount int) {
	currentFilename := fmt.Sprintf("%s_%d", prefix, count)
	nextFilename := fmt.Sprintf("%s_%d", prefix, nextCount)

	// Open the current file for appending
	file, err := os.OpenFile(currentFilename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Printf("Error opening file for appending: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Write the checksum and reference to the next split file
	content := fmt.Sprintf("Checksum: %s\nNextFile: %s\n", checksum, nextFilename)
	_, err = file.WriteString(content)
	if err != nil {
		fmt.Printf("Error writing to file: %s\n", err)
		os.Exit(1)
	}
}
