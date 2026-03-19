package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./scripts/json_validator.go <path-to-json-file>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Read the JSON file
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		os.Exit(1)
	}

	// Validate JSON
	var jsonData interface{}
	if err := json.Unmarshal(fileContent, &jsonData); err != nil {
		fmt.Printf("Invalid JSON: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("The JSON is valid.")
	os.Exit(0)
}
