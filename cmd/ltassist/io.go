package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	re = regexp.MustCompile(`(#+) (\S+)`) // find header
	rs = regexp.MustCompile(`\*\S+\*$`)   // find data type
)

type fieldDoc struct {
	text     string
	dataType string
}

// TODO: find docs with proper markdown ast library :)
func findDoc(name, docPath string) (*fieldDoc, error) {
	file, err := os.Open(docPath)
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(file)
	var doc, dataType string
	for s.Scan() {
		if re.MatchString(s.Text()) {
			sm := re.FindStringSubmatch(s.Text())[2]
			if strings.EqualFold(sm, name) {
				for s.Scan() {
					text := s.Text()
					if strings.HasPrefix(text, "#") {
						break
					}
					if rs.MatchString(text) {
						dataType = strings.ReplaceAll(rs.FindString(text), "*", "")
						continue
					}
					doc += text + "\n"
				}
			}
		}
	}
	return &fieldDoc{
		text:     strings.TrimSpace(doc),
		dataType: dataType,
	}, nil
}

// get user input
func readInput(name, kind string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\033[1;33m%s\033[0m (%s): ", name, kind)
	text, _ := reader.ReadString('\n')
	return text
}
