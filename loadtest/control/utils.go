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
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
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
	emojis            = []string{":grinning:", ":slightly_smiling_face:", ":smile:", ":sunglasses:", ":innocent:", ":hugging_face:", ":shushing_face:", ":face_with_finger_covering_closed_lips:", ":thinking_face:", ":thinking:", ":zipper_mouth_face:", ":face_with_raised_eyebrow:", ":face_with_one_eyebrow_raised:", ":neutral_face:", ":expressionless:", ":no_mouth:", ":smirk:", ":unamused:", ":face_with_rolling_eyes:", ":roll_eyes:", ":grimacing:", ":lying_face:", ":relieved:", ":pensive:", ":sleepy:", ":drooling_face:", ":sleeping:", ":mask:", ":give_back_money:"} // The last one is a custom emoji

	serverVersionRE = regexp.MustCompile(`\d+.\d+\.\d+`)
	links           = []string{
		"https://github.com/mattermost/mattermost",
		"https://www.youtube.com/watch?v=-5jompL6G-k",
		"https://www.youtube.com/watch?v=GKLyAVHgNzY",
		"https://mattermost.com",
		"https://developers.mattermost.com",
		"https://golang.org",
		"https://reactjs.org",
	}
	// MinSupportedVersion is, by definition, the third-to-last ESR
	MinSupportedVersion = semver.MustParse("7.8.0")

	// UnreleasedVersion is a version guaranteed to be larger than any released
	// version, useful for actions already added to the load-test but not yet
	// merged in the server.
	UnreleasedVersion = semver.Version{Major: math.MaxUint64}
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

// RandomEmoji returns a random emoji from a list.
func RandomEmoji() string {
	return emojis[rand.Intn(len(emojis))]
}

// AddLink appends a link to a string to test the LinkPreview feature.
func AddLink(input string) string {
	link := RandomLink()

	return input + " " + link + " "
}

func RandomLink() string {
	n := rand.Int() % len(links)
	link := links[n]

	return link
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
	return PickRandomString(words)
}

// PickRandomString returns a random string from the given slice of strings
func PickRandomString(strings []string) string {
	return strings[rand.Intn(len(strings))]
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

// ParseServerVersion finds the semver-compatible version substring in the string
// returned by the server and tries to parse it
func ParseServerVersion(versionString string) (semver.Version, error) {
	serverVersion := serverVersionRE.FindString(versionString)
	return semver.Parse(serverVersion)
}

// AttachFilesToPost uploads at least one file on behalf of the user, attaching
// all uploaded files to the post.
func AttachFilesToPost(u user.User, post *model.Post) error {
	fileIDs, err := _attachFilesToObj(u, post.ChannelId)
	if err != nil {
		return err
	}
	post.FileIds = fileIDs
	return nil
}

// AttachFilesToDraft uploads at least one file on behalf of the user, attaching
// all uploaded files to the draft.
func AttachFilesToDraft(u user.User, draft *model.Draft) error {
	fileIDs, err := _attachFilesToObj(u, draft.ChannelId)
	if err != nil {
		return err
	}
	draft.FileIds = fileIDs
	return nil
}

func AttachFileToBookmark(u user.User, bookmark *model.ChannelBookmark) error {
	filenames := []string{"test_upload.png", "test_upload.jpg", "test_upload.mp4", "test_upload.txt"}
	file := filenames[rand.Intn(len(filenames))]
	data := MustAsset(file)
	resp, err := u.UploadFile(data, bookmark.ChannelId, file)
	if err != nil {
		return err
	}

	bookmark.FileId = resp.FileInfos[0].Id
	return nil
}

func _attachFilesToObj(u user.User, channelID string) ([]string, error) {
	type file struct {
		data   []byte
		upload bool
	}
	filenames := []string{"test_upload.png", "test_upload.jpg", "test_upload.mp4", "test_upload.txt"}
	files := make(map[string]*file, len(filenames))

	for _, filename := range filenames {
		files[filename] = &file{
			data:   MustAsset(filename),
			upload: rand.Intn(2) == 0,
		}
	}

	// We make sure at least one file gets uploaded.
	files[filenames[rand.Intn(len(filenames))]].upload = true

	var wg sync.WaitGroup
	fileIdsChan := make(chan string, len(files))
	errChan := make(chan error, len(files))
	for filename, file := range files {
		if !file.upload {
			continue
		}
		wg.Add(1)
		go func(filename string, data []byte) {
			defer wg.Done()
			resp, err := u.UploadFile(data, channelID, filename)
			if err != nil {
				errChan <- err
				return
			}
			fileIdsChan <- resp.FileInfos[0].Id
		}(filename, file.data)
	}

	wg.Wait()
	close(fileIdsChan)
	close(errChan)

	// Attach all successfully uploaded files
	var fileIDs []string
	for fileId := range fileIdsChan {
		fileIDs = append(fileIDs, fileId)
	}

	// Collect all errors
	var finalErr error
	for err := range errChan {
		finalErr = errors.Join(finalErr, err)
	}

	return fileIDs, finalErr
}
