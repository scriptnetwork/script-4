package common

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

    log "github.com/sirupsen/logrus"
)

// reward distribution algorithm
var Height_hf1 uint64
var Height_hf2 uint64 //tokenomics v2. Licenses

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "hf"})

// Exported function to initialize variables from a file
func Initialize_hf_values(file_path string) error {
	logger.Info("Initialize_hf_values")

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
    	logger.Infof("line %v", line)
		if len(parts) == 2 {
            key := strings.TrimSpace(parts[0])
			value, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing value: %w", err)
			}
	        if key == "hf1" {
				Height_hf1 = value
			} else if key == "hf2" {
				Height_hf2 = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}


