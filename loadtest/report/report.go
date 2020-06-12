// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"encoding/json"
	"io/ioutil"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
)

// Generator is used to generate load test reports.
type Generator struct {
	label  string
	helper *prometheus.Helper
}

// New returns a new instance of a generator.
func New(label string, helper *prometheus.Helper) *Generator {
	return &Generator{
		label:  label,
		helper: helper,
	}
}

// Load loads a report from a given file path.
func Load(path string) (Report, error) {
	var r Report
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(buf, &r)
	if err != nil {
		return r, err
	}
	return r, nil
}
