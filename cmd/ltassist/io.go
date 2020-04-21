package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

const moveLineUp = "\033[1A\033[2K\r"

var (
	headerRegex = regexp.MustCompile(`(#+) (\S+)`) // find header
	typeRegex   = regexp.MustCompile(`\*\S+\*$`)   // find data type
)

type fieldDoc struct {
	text     string
	dataType string
}

// TODO: find docs with proper markdown ast library
func findDoc(name, docPath string) (*fieldDoc, error) {
	file, err := os.Open(docPath)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(file)
	var doc, dataType string
	for s.Scan() {
		if headerRegex.MatchString(s.Text()) {
			sm := headerRegex.FindStringSubmatch(s.Text())[2]
			if strings.EqualFold(sm, name) {
				for s.Scan() {
					text := s.Text()
					if strings.HasPrefix(text, "#") {
						break
					}
					if typeRegex.MatchString(text) {
						dataType = strings.Trim(typeRegex.FindString(text), "*")
						continue
					}
					doc += text + "\n"
				}
			}
		}
	}
	if s.Err() != nil {
		return nil, err
	}

	return &fieldDoc{
		text:     strings.TrimSpace(doc),
		dataType: dataType,
	}, nil
}

// get user input, returns default on empty input
func readInput(kind, defaultValue string) (string, error) {
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (%s): ", color.YellowString("Enter a value"), kind)
	text, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if text == "\n" {
		text = defaultValue
		fmt.Printf("%s%s (%s): %s\n", moveLineUp, color.YellowString("Enter a value"), kind, text)
	}
	fmt.Println()
	return text, nil
}
