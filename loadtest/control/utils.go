package control

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	pkgPath = "github.com/mattermost/mattermost-load-test-ng/loadtest/"
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var re = regexp.MustCompile(`-[[:alpha:]]+`)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetErrOrigin() string {
	var origin string
	if pc, file, line, ok := runtime.Caller(2); ok {
		if f := runtime.FuncForPC(pc); f != nil {
			if wd, err := os.Getwd(); err == nil {
				origin = fmt.Sprintf("%s %s:%d", strings.TrimPrefix(f.Name(), pkgPath), strings.TrimPrefix(file, wd+string(os.PathSeparator)), line)
			}
		}
	}
	return origin
}

// assuming the incoming name has a pattern of {{agent-id}}-{{user-name}}-{{user-number}}
func RandomizeUserName(name string) string {
	parts := re.FindAllString(name, -1)
	if len(parts) > 0 {
		random := letters[rand.Intn(len(letters))]
		name = strings.Replace(name, parts[len(parts)-1], "-user"+string(random), 1)
	}
	return name
}
