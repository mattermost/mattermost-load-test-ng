// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

type Configuration struct {
	PrometheusURL                 string
	MetricsUpdateIntervalInMS     int
	HealthcheckUpdateIntervalInMS int
}

type Query struct {
	Description string
	Query       string
	Threshold   float64
	Alert       bool
}
