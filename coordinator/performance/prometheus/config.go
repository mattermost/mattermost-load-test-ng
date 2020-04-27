// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

type Configuration struct {
	PrometheusURL                 string
	MetricsUpdateIntervalInMS     int
	HealthcheckUpdateIntervalInMS int
}

type Query struct {
	Description string  `default:"Request duration" validate:"text"`
	Query       string  `default:"rate(mattermost_http_request_duration_seconds_sum[1m])/rate(mattermost_http_request_duration_seconds_count[1m])" validate:"text"`
	Threshold   float64 `default:"0.2" validate:"range:[0,]"`
	Alert       bool    `default:"true"`
}
