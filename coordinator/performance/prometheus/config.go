// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

type Configuration struct {
	PrometheusURL                 string `default:"http://localhost:9090" validate:"url"`
	MetricsUpdateIntervalInMS     int    `default:"1000" validate:"range:[0,]"`
	HealthcheckUpdateIntervalInMS int    `default:"60000" validate:"range:[0,]"`
}

// Query contains the needed information to perform a query.
type Query struct {
	// The description for the query.
	Description string `validate:"notempty"`
	// An optional string for populating the legend of this query's panel
	// in the Grafana dashboard containing all coordinator metrics
	Legend string
	// The PromQL query to be run.
	Query string `validate:"notempty"`
	// The value over which the performance monitor will fire an alert
	// to the coordinator's feedback loop.
	Threshold float64 `validate:"range:[0,]"`
	// The minimum amount of time (in seconds) that needs to have passed
	// since the start of the monitoring process before the query can be run.
	MinIntervalSec int `validate:"range:[0,]"`
	// The value indicating whether or not to fire an alert.
	Alert bool
}
