// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ltmbebootstrap implements the WS4(a) MBE setup phase: it grants Encryption
// Manager (EM) to the sysadmin account, creates a pool of MBE channels, and
// throttled-adds existing dump users to them as channel_admin. It is run
// out-of-band, before a measured load-test run, against a standalone
// deployment (see mbe-load-test-plan.md WS4/M1/M3).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

type mbeChannel struct {
	Id          string `json:"id"`
	TeamId      string `json:"team_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func main() {
	var (
		serverURL         = flag.String("server-url", "", "Mattermost server URL, e.g. http://1.2.3.4:8065 (required)")
		adminEmail        = flag.String("admin-email", "sysadmin@sample.mattermost.com", "sysadmin email/login id")
		adminPassword     = flag.String("admin-password", "Sys@dmin-sample1", "sysadmin password")
		teamName          = flag.String("team", "", "name of the team to create MBE channels in (required)")
		pluginID          = flag.String("plugin-id", "message-based-encryption", "MBE plugin id")
		numChannels       = flag.Int("channels", 2, "number of MBE channels to create")
		channelPrefix     = flag.String("channel-prefix", "mbe-bootstrap", "channel name prefix")
		membersPerChannel = flag.Int("members-per-channel", 100, "number of existing dump users to add to each channel")
		addsPerSecond     = flag.Float64("rate", 4, "max total channel-member-adds per second across all channels")
		usersPageSize     = flag.Int("users-page-size", 200, "page size when listing candidate users")
		outputFile        = flag.String("output", "mbe-channels.json", "path to write the created channel-ID list (JSON)")
		existingChannels  = flag.String("existing-channels", "", "path to a channel-ID JSON file from a previous run; when set, reuses those channels instead of creating new ones (for adding a fresh batch of members, e.g. simulcontroller users created after the first bootstrap pass)")
		usernamePrefix    = flag.String("username-prefix", "", "when set, find candidate members by username prefix (e.g. a load-test agent's '<cluster>-agent-N-' pattern) via user search, instead of paging through all users")
	)
	flag.Parse()

	if *serverURL == "" || *teamName == "" {
		fmt.Fprintln(os.Stderr, "-server-url and -team are required")
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*serverURL, *adminEmail, *adminPassword, *teamName, *pluginID, *numChannels, *channelPrefix, *membersPerChannel, *addsPerSecond, *usersPageSize, *outputFile, *existingChannels, *usernamePrefix); err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
}

func run(serverURL, adminEmail, adminPassword, teamName, pluginID string, numChannels int, channelPrefix string, membersPerChannel int, addsPerSecond float64, usersPageSize int, outputFile, existingChannelsFile, usernamePrefix string) error {
	ctx := context.Background()
	client := model.NewAPIv4Client(serverURL)

	me, _, err := client.Login(ctx, adminEmail, adminPassword)
	if err != nil {
		return fmt.Errorf("login as %s: %w", adminEmail, err)
	}
	log.Printf("logged in as %s (id=%s)", adminEmail, me.Id)

	if _, _, err := pluginRequest(ctx, client, pluginID, http.MethodPost, "/encryption-manager/"+me.Id, nil); err != nil {
		return fmt.Errorf("grant encryption manager to %s: %w", me.Id, err)
	}
	log.Printf("granted Encryption Manager to %s", me.Id)

	if _, body, err := pluginRequest(ctx, client, pluginID, http.MethodPost, "/test-connection", nil); err != nil {
		return fmt.Errorf("test-connection: %w", err)
	} else {
		log.Printf("test-connection ok: %s", strings.TrimSpace(string(body)))
	}

	team, _, err := client.GetTeamByName(ctx, teamName, "")
	if err != nil {
		return fmt.Errorf("get team %q: %w", teamName, err)
	}

	if _, _, err := client.AddTeamMember(ctx, team.Id, me.Id); err != nil {
		log.Printf("add %s to team %s: %v (continuing — likely already a member)", me.Id, teamName, err)
	} else {
		log.Printf("added %s to team %s", me.Id, teamName)
	}

	needed := numChannels * membersPerChannel
	var candidates []string
	if usernamePrefix != "" {
		candidates, err = collectCandidateUsersByPrefix(ctx, client, usernamePrefix, needed)
	} else {
		candidates, err = collectCandidateUsers(ctx, client, me.Id, needed, usersPageSize)
	}
	if err != nil {
		return fmt.Errorf("collect candidate users: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("found 0 candidate users; nothing to add")
	}
	if len(candidates) < needed {
		log.Printf("found %d candidate users, wanted %d distinct; reusing users across channels to fill every channel's quota (each channel still gets %d members, just not all distinct server-wide)", len(candidates), needed, membersPerChannel)
	}

	var channels []mbeChannel
	if existingChannelsFile != "" {
		data, err := os.ReadFile(existingChannelsFile)
		if err != nil {
			return fmt.Errorf("read existing channels file %s: %w", existingChannelsFile, err)
		}
		if err := json.Unmarshal(data, &channels); err != nil {
			return fmt.Errorf("parse existing channels file %s: %w", existingChannelsFile, err)
		}
		log.Printf("reusing %d existing channels from %s", len(channels), existingChannelsFile)
	} else {
		channels = make([]mbeChannel, 0, numChannels)
		for i := 0; i < numChannels; i++ {
			name := fmt.Sprintf("%s-%d", channelPrefix, i)
			displayName := fmt.Sprintf("MBE Bootstrap Channel %d", i)
			body := map[string]string{
				"team_id":      team.Id,
				"name":         name,
				"display_name": displayName,
			}
			_, respBody, err := pluginRequest(ctx, client, pluginID, http.MethodPost, "/channels", body)
			if err != nil {
				return fmt.Errorf("create channel %s: %w", name, err)
			}
			var ch model.Channel
			if err := json.Unmarshal(respBody, &ch); err != nil {
				return fmt.Errorf("parse create-channel response for %s: %w", name, err)
			}
			channels = append(channels, mbeChannel{Id: ch.Id, TeamId: ch.TeamId, Name: ch.Name, DisplayName: ch.DisplayName})
			log.Printf("created MBE channel %s (id=%s)", name, ch.Id)
		}
	}

	interval := time.Duration(float64(time.Second) / addsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// pos advances globally across all channels and wraps via modulo once the candidate pool is
	// exhausted, so every channel still gets its full membersPerChannel quota even when the server
	// has fewer distinct users than numChannels*membersPerChannel. This assumes
	// len(candidates) > membersPerChannel, so a wraparound repeat always lands in a different
	// channel than the user's first occurrence (never a duplicate add within the same channel).
	var succeeded, failed int
	pos := 0
	for _, ch := range channels {
		want := membersPerChannel
		for n := 0; n < want; n++ {
			userID := candidates[pos%len(candidates)]
			pos++
			<-ticker.C
			// Join the team first: candidates come from the whole dump, not just team.Id, so most
			// won't be team members yet. AddTeamMember errors (e.g. already a member) are swallowed
			// here — a real problem (deactivated user, etc.) will still surface as an AddChannelMember
			// failure below, which is what gets counted and logged.
			if _, _, err := client.AddTeamMember(ctx, team.Id, userID); err != nil {
				log.Printf("add %s to team %s: %v (continuing — likely already a member)", userID, teamName, err)
			}
			if _, _, err := client.AddChannelMember(ctx, ch.Id, userID); err != nil {
				failed++
				log.Printf("add member %s to channel %s failed: %v", userID, ch.Name, err)
				continue
			}
			succeeded++
			if succeeded%25 == 0 {
				log.Printf("progress: %d members added so far", succeeded)
			}
		}
	}
	log.Printf("member-add complete: %d succeeded, %d failed", succeeded, failed)

	out, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	if err := os.WriteFile(outputFile, out, 0o644); err != nil {
		return fmt.Errorf("write output file %s: %w", outputFile, err)
	}
	log.Printf("wrote %d channel IDs to %s", len(channels), outputFile)

	if failed > 0 {
		return fmt.Errorf("%d member-add failures — see log above", failed)
	}
	return nil
}

// collectCandidateUsers pages through GET /api/v4/users as sysadmin and returns up to `needed`
// distinct user IDs, excluding excludeID (the acting EM/sysadmin itself). Candidates are drawn from
// the whole dump, not scoped to the target team — capping the pool at existing team members risks
// running out of candidates if the team is smaller than needed. The caller joins each candidate to
// the team before adding it to a channel.
func collectCandidateUsers(ctx context.Context, client *model.Client4, excludeID string, needed, pageSize int) ([]string, error) {
	var ids []string
	for page := 0; len(ids) < needed; page++ {
		users, _, err := client.GetUsers(ctx, page, pageSize, "")
		if err != nil {
			return nil, fmt.Errorf("get users page %d: %w", page, err)
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			if u.Id == excludeID {
				continue
			}
			ids = append(ids, u.Id)
			if len(ids) >= needed {
				break
			}
		}
		if len(users) < pageSize {
			break
		}
	}
	return ids, nil
}

// collectCandidateUsersByPrefix finds up to `needed` user IDs whose username starts with prefix,
// via user search. Used to target simulcontroller-created users (e.g. "<cluster>-agent-0-"), which
// don't exist yet at the time of an initial bootstrap pass against the base dump and so can't be
// found by collectCandidateUsers. Not scoped to the team, for the same reason as
// collectCandidateUsers — the caller joins each candidate to the team before adding it to a channel.
func collectCandidateUsersByPrefix(ctx context.Context, client *model.Client4, prefix string, needed int) ([]string, error) {
	users, _, err := client.SearchUsers(ctx, &model.UserSearch{Term: prefix, AllowInactive: true, Limit: needed})
	if err != nil {
		return nil, fmt.Errorf("search users with prefix %q: %w", prefix, err)
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		if !strings.HasPrefix(u.Username, prefix) {
			continue
		}
		ids = append(ids, u.Id)
		if len(ids) >= needed {
			break
		}
	}
	return ids, nil
}

// pluginRequest issues an authenticated request against a plugin's own HTTP route
// (base URL + "/plugins/<pluginID><path>"), which sits outside the core /api/v4 tree that
// model.Client4's DoAPIRequest* helpers assume, so the request is built by hand.
func pluginRequest(ctx context.Context, client *model.Client4, pluginID, method, path string, body any) (*http.Response, []byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reader = bytes.NewReader(b)
	}

	url := strings.TrimRight(client.URL, "/") + "/plugins/" + pluginID + path
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(model.HeaderAuth, client.AuthType+" "+client.AuthToken)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return resp, respBody, errors.New(resp.Status + ": " + strings.TrimSpace(string(respBody)))
	}
	return resp, respBody, nil
}
