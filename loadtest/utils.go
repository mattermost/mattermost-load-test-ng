// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

func pickRate(config UserControllerConfiguration) (float64, error) {
	dist := config.RatesDistribution
	if len(dist) == 0 {
		return 1.0, nil
	}

	weights := make([]int, len(dist))
	for i := range dist {
		weights[i] = int(dist[i].Percentage * 100)
	}

	idx, err := control.SelectWeighted(weights)
	if err != nil {
		return -1, fmt.Errorf("loadtest: failed to select weight: %w", err)
	}

	return dist[idx].Rate, nil
}
