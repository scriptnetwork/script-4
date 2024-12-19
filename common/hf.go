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
		// Check if the line contains the parameter hf1
		if strings.HasPrefix(line, "hf1") {
			// Split the line on '='
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Convert the value to an integer
				value, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
				if err != nil {
					return fmt.Errorf("error parsing hf1 value: %w", err)
				}
				Height_hf1 = value
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}


