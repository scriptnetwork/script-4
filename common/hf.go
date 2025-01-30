package common

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// reward distribution algorithm
var Height_hf1 uint64
var Height_hf2 uint64 //tokenomics v2. Licenses

// Exported function to initialize variables from a file
func Initialize_hf_values(file_path string) error {
	// Open the file
	file, err := os.Open(file_path)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
	        // Check if the line contains the parameter hf1
			value, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing value: %w", err)
			}
	        if parts[0] == "hf1" {
				Height_hf1 = value
			} else if parts[0] == "hf2" {
				Height_hf2 = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}


