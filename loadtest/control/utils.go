// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

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

// TODO: this is currently unused. Should be probably called once when starting
// the load-test cmd or API server. It should also be called only when running
// a load-test in production.
func init() {
	rand.Seed(time.Now().UnixNano())
}

// getErrOrigin returns a string indicating the location of the error that
// just occurred. It's a utility function used to find out exactly where an
// error has happened since actions inside a UserController might get called
// from a single place.
func getErrOrigin() string {
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

// RandomizeUserName is a utility function used by UserController's implementations
// to randomize a username while keeping a basic pattern unchanged.
// Assumes the given name has a pattern of {{agent-id}}-{{user-name}}-{{user-number}}.
// If the pattern is not found it will return the input string unaltered.
func RandomizeUserName(name string) string {
	parts := re.FindAllString(name, -1)
	if len(parts) > 0 {
		random := letters[rand.Intn(len(letters))]
		name = strings.Replace(name, parts[len(parts)-1], "-user"+string(random), 1)
	}
	return name
}

func emulateUserTyping(t string, cb func(term string) UserActionResponse) UserActionResponse {
	typingSpeed := time.Duration(100+rand.Intn(20)*10) * time.Millisecond // 100-300ms
	typeCorrect := time.Tick(typingSpeed)
	typeWrong := time.Tick(time.Duration(2+rand.Intn(8)) * typingSpeed)
	runes := []rune(t)
	var term string
	var resp UserActionResponse
	for i := 0; i < len(runes); {
		select {
		case <-typeCorrect:
			term += string(runes[i])
			resp = cb(term)
			if resp.Err != nil {
				return resp
			}
			i++
		case <-typeWrong:
			resp = cb(term + "a")
			if resp.Err != nil {
				return resp
			}
			time.Sleep(100 * time.Millisecond)
			resp = cb(term)
			if resp.Err != nil {
				return resp
			}
		}
	}
	return resp
}
