/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	FixedBlockSize = 48 << 10 // according to element limit
)

func main() {
	// Command-line flags
	var inputFile string
	var outputPrefix string
	var chunkSize int
	wd, _ := os.Getwd()
	inputFilePath := wd + "/examples/compression/test.pdf"

	flag.StringVar(&inputFile, "input", inputFilePath, "Input file to be split")
	flag.StringVar(&outputPrefix, "output-prefix", "output", "Output file prefix")
	flag.IntVar(&chunkSize, "chunk-size", FixedBlockSize, "Size of each chunk in lines")
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

	// Create and open the first output file
	outputFile, err := createOutputFile(outputPrefix, chunkCount)
	if err != nil {
		fmt.Printf("Error creating output file: %s\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	for {
		// Read a chunk of data from the input file
		n, err := input.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Printf("Error reading input file: %s\n", err)
			os.Exit(1)
		}

		// Write the chunk to the current output file
		_, err = outputFile.Write(buf[:n])
		if err != nil {
			fmt.Printf("Error writing to output file: %s\n", err)
			os.Exit(1)
		}

		currentChunkSize += n

		// Check if the current chunk size exceeds the specified chunk size
		if currentChunkSize >= chunkSize {
			// Close the current output file
			outputFile.Close()

			// Increment the chunk count
			chunkCount++

			// Create and open the next output file
			outputFile, err = createOutputFile(outputPrefix, chunkCount)
			if err != nil {
				fmt.Printf("Error creating output file: %s\n", err)
				os.Exit(1)
			}

			// Reset the current chunk size
			currentChunkSize = 0
		}

		// Break the loop if we've reached the end of the input file
		if err == io.EOF {
			break
		}
	}

	fmt.Println("Split operation completed successfully.")
}

// createOutFile creates a new output file with the specified prefix and chunk count.
func createOutputFile(prefix string, count int) (*os.File, error) {
	filename := fmt.Sprintf("%s_%d", prefix, count)
	return os.Create(filename)
}
