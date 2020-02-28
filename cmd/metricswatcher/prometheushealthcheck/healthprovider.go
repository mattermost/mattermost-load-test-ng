// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheushealthcheck

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

type HealthProvider struct {
	healthEndpoint *url.URL
}

type HealthCheckResult struct {
	Healthy   bool
	Error     error
	Timestamp time.Time
}

func NewHealthProvider(server string) (*HealthProvider, error) {
	url, err := url.Parse(server)

	if err != nil {
		return nil, fmt.Errorf("error while starting healthcheck: %w", err)
	}

	url.Path = path.Join(url.Path, "/-/healthy")

	return &HealthProvider{url}, nil
}

func (h *HealthProvider) Check() HealthCheckResult {
	response, err := http.Get(h.healthEndpoint.String())
	now := time.Now()

	if err != nil {
		return HealthCheckResult{
			Healthy:   false,
			Error:     err,
			Timestamp: now,
		}
	}

	healthy := response.StatusCode == http.StatusOK

	if !healthy {
		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)
		err = errors.New(string(body))
	}

	return HealthCheckResult{
		Healthy:   healthy,
		Error:     err,
		Timestamp: now,
	}
}
