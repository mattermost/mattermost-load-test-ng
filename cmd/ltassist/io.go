package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
	"unsafe"
)

var (
	re = regexp.MustCompile(`(#+) (\S+)`) // find header
	rs = regexp.MustCompile(`\*\S+\*$`)   // find data type
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

type fieldDoc struct {
	text     string
	dataType string
}

func findDoc(name string) (*fieldDoc, error) {
	file, err := os.Open(docFile)
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
					doc += text
				}
			}
		}
	}
	return &fieldDoc{
		text:     doc,
		dataType: dataType,
	}, nil
}

func readInput(name, kind string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (%s): ", name, kind)
	text, _ := reader.ReadString('\n')
	return text
}

func rewind(n int) {
	// TODO: check for non-POSIX systems
	rewind := []byte("\033[1A\033[2K\r")
	for i := 0; i < n; i++ {
		fmt.Printf("%s", rewind)
	}
}

// we are not printing anything larger than terminal width
// so that we can safely rewind and cleanup w/o mess.
func adjustToWindow(s string) string {
	tl := len(s)
	tw := getWidth()
	if tl >= tw {
		runes := []rune(s)
		for i := tw; i <= tl; i += tw {
			var r rune
			runes = append(runes, r)
			copy(runes[i+1:], runes[i:])
			runes[i] = '\n'
		}
		return strings.TrimSpace(string(runes))
	}
	return s
}

func getWidth() int {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return int(ws.Col)
}
