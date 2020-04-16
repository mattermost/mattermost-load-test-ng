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

// TODO: find docs with proper markdown ast library
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
						dataType = strings.Trim(rs.FindString(text), "*")
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

// get user input, returns default on empty input
func readInput(kind, defaultValue string) string {
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (%s): ", yellow("Enter a value"), kind)
	text, err := r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	if text == "\n" {
		text = defaultValue
		moveLineUp := "\033[1A\033[2K\r"
		fmt.Printf("%s%s (%s): %s\n", moveLineUp, yellow("Enter a value"), kind, text)
	}
	fmt.Println()
	return text
}

func yellow(v ...interface{}) string {
	return fmt.Sprintf("\033[1;33m%s\033[0m", v...)
}

func green(v ...interface{}) string {
	return fmt.Sprintf("\033[1;32m%s\033[0m", v...)
}

func red(v ...interface{}) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", v...)
}
