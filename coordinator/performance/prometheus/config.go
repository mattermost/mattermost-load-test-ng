// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

type Configuration struct {
	PrometheusURL                 string `default:"http://localhost:9090" validate:"url"`
	MetricsUpdateIntervalInMS     int    `default:"1000" validate:"range:[0,]"`
	HealthcheckUpdateIntervalInMS int    `default:"60000" validate:"range:[0,]"`
}

type Query struct {
	Description string  `default:"Request duration" validate:"notempty"`
	Query       string  `default:"sum(increase(mattermost_api_time_sum[1m])) by (instance) / sum(increase(mattermost_api_time_count[1m])) by (instance)" validate:"notempty"`
	Threshold   float64 `default:"0.2" validate:"range:[0,]"`
	Alert       bool    `default:"true"`
}
