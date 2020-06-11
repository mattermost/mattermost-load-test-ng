// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

type Configuration struct {
	PrometheusURL                 string `default:"http://localhost:9090" validate:"url"`
	MetricsUpdateIntervalInMS     int    `default:"1000" validate:"range:[0,]"`
	HealthcheckUpdateIntervalInMS int    `default:"60000" validate:"range:[0,]"`
}

type Query struct {
	Description string  `validate:"notempty"`
	Query       string  `validate:"notempty"`
	Threshold   float64 `validate:"range:[0,]"`
	Alert       bool
}
