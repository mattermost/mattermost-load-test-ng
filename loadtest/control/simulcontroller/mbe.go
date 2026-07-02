// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"encoding/json"
	"fmt"
	"os"
)

// mbeChannel mirrors the JSON shape written by cmd/ltmbebootstrap's -output file.
type mbeChannel struct {
	Id string `json:"id"`
}

// LoadMBEChannelIDs reads the MBE channel-ID list emitted by cmd/ltmbebootstrap. An empty path
// returns a nil slice and no error, so callers can pass Config.MBEChannelIdsFile unconditionally.
func LoadMBEChannelIDs(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read MBE channel-ids file %s: %w", path, err)
	}

	var channels []mbeChannel
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("parse MBE channel-ids file %s: %w", path, err)
	}

	ids := make([]string, len(channels))
	for i, c := range channels {
		ids[i] = c.Id
	}

	return ids, nil
}
