/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func compressAndSplit(inputFilePath, outputArchivePath string, chunkSizeKB int) error {
	cmd := exec.Command("7z", "a", fmt.Sprintf("-v%dk", chunkSizeKB), outputArchivePath, inputFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to compress and split: %v", err)
	}

	return nil
}

func main() {
	wd, _ := os.Getwd()

	inputFilePath := wd + "/examples/compression/test.pdf"
	outputArchivePath := "output_archive.7z"
	chunkSizeMB := 40

	err := compressAndSplit(inputFilePath, outputArchivePath, chunkSizeMB)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Compression and splitting completed.")
}
