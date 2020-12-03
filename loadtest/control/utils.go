//go:generate go-bindata -nometadata -mode 0644 -pkg control -o ./bindata.go -prefix "../../testdata/" ../../testdata/
// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
)

type PostsSearchOpts struct {
	From     string
	In       string
	On       time.Time
	Before   time.Time
	After    time.Time
	Excluded []string
	IsPhrase bool
}

func init() {
	words = strings.Split(MustAssetString("test_text.txt"), "\n")
}

const (
	pkgPath = "github.com/mattermost/mattermost-load-test-ng/loadtest/"
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	userNameRe        = regexp.MustCompile(`-[[:alpha:]]+`)
	teamDisplayNameRe = regexp.MustCompile(`team[0-9]+(.*)`)
	words             = []string{}
	emojis            = []string{":grinning:", ":slightly_smiling_face:", ":smile:", ":sunglasses:"}
	serverVersionRE   = regexp.MustCompile(`\d+.\d+\.\d+`)
)

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
	parts := userNameRe.FindAllString(name, -1)
	if len(parts) > 0 {
		random := letters[rand.Intn(len(letters))]
		name = strings.Replace(name, parts[len(parts)-1], "-user"+string(random), 1)
	}
	return name
}

// RandomizeTeamDisplayName is a utility function to set a random team display name
// while keeping the basic pattern unchanged.
// Assumes the given name has a pattern of team{{number}}[-letter].
func RandomizeTeamDisplayName(name string) string {
	matches := teamDisplayNameRe.FindStringSubmatch(name)
	if len(matches) == 2 {
		name = matches[0] + "-" + string(letters[rand.Intn(len(letters))])
	}
	return name
}

// EmulateUserTyping calls cb function for each rune in the input string.
func EmulateUserTyping(t string, cb func(term string) UserActionResponse) UserActionResponse {
	typingSpeed := time.Duration(100+rand.Intn(200)) * time.Millisecond // 100-300ms

	runes := []rune(t)
	var term string
	var resp UserActionResponse
	for i := 0; i < len(runes); i++ {
		time.Sleep(typingSpeed)
		term += string(runes[i])
		resp = cb(term)
		if resp.Err != nil {
			return resp
		}
		// 0.15% probability of mistyping. Add a rune which will be overridden
		// by next iteration.
		if rand.Float32() < 0.15 && i < len(runes)-1 {
			time.Sleep(typingSpeed)
			resp = cb(term + "a")
			if resp.Err != nil {
				return resp
			}
		}
	}
	return resp
}

// GenerateRandomSentences generates random string from test_text file.
func GenerateRandomSentences(count int) string {
	if count <= 0 {
		return "🙂" // if there is nothing to say, an emoji worths for thousands
	}

	var withEmoji bool
	// 10% of the times we add an emoji to the message.
	if rand.Float64() < 0.10 {
		withEmoji = true
		count--
	}

	var random string
	for i := 0; i < count; i++ {
		n := rand.Int() % len(words)
		random += words[n] + " "
	}

	if withEmoji {
		return random + emojis[rand.Intn(len(emojis))]
	}

	return random[:len(random)-1] + "."
}

// SelectWeighted does a random weighted selection on a given slice of weights.
func SelectWeighted(weights []int) (int, error) {
	var sum int
	if len(weights) == 0 {
		return -1, errors.New("weights cannot be empty")
	}
	for i := range weights {
		sum += weights[i]
	}
	if sum == 0 {
		return -1, errors.New("weights frequency sum cannot be zero")
	}
	distance := rand.Intn(sum)
	for i := range weights {
		distance -= weights[i]
		if distance < 0 {
			return i, nil
		}
	}
	return -1, errors.New("should not be able to reach this point")
}

// PickRandomWord returns a random  word.
func PickRandomWord() string {
	return words[rand.Intn(len(words))]
}

// GeneratePostsSearchTerm generates a posts search term from the given
// words and options.
func GeneratePostsSearchTerm(words []string, opts PostsSearchOpts) string {
	var term string

	if opts.From != "" {
		term += fmt.Sprintf("from:%s ", opts.From)
	}

	if opts.In != "" {
		term += fmt.Sprintf("in:%s ", opts.In)
	}

	if !opts.On.IsZero() {
		term += fmt.Sprintf("on:%s ", opts.On.Format("2006-01-02"))
	}

	if !opts.Before.IsZero() {
		term += fmt.Sprintf("before:%s ", opts.Before.Format("2006-01-02"))
	}

	if !opts.After.IsZero() {
		term += fmt.Sprintf("after:%s ", opts.After.Format("2006-01-02"))
	}

	if len(opts.Excluded) > 0 {
		for _, w := range opts.Excluded {
			term += fmt.Sprintf("-%s ", w)
		}
	}

	if opts.IsPhrase {
		term += "\"" + strings.Join(words, " ") + "\""
	} else {
		term += strings.Join(words, " ")
	}

	return term
}

func PickIdleTimeMs(minIdleTimeMs, avgIdleTimeMs int, rate float64) time.Duration {
	// Randomly selecting a value in the interval
	// [minIdleTimeMs, avgIdleTimeMs*2 - minIdleTimeMs).
	// This will give us an expected value equal to avgIdleTimeMs.
	// TODO: consider if it makes more sense to select this value using
	// a truncated normal distribution.
	idleMs := rand.Intn(avgIdleTimeMs*2-minIdleTimeMs*2) + minIdleTimeMs
	idleTimeMs := time.Duration(math.Round(float64(idleMs) * rate))

	return idleTimeMs * time.Millisecond
}

// IsVersionSupported returns whether a given version is supported
// by the provided server version string.
func IsVersionSupported(version, serverVersionString string) (bool, error) {
	v, err := semver.Parse(version)
	if err != nil {
		return false, err
	}

	serverVersion := serverVersionRE.FindString(serverVersionString)

	sv, err := semver.Parse(serverVersion)
	if err != nil {
		return false, err
	}

	return v.LTE(sv), nil
}
