// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

import (
	"context"
	"time"

	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// API is a subset of Prometheus API interface.
// https://github.com/prometheus/client_golang/blob/v1.25.0/api/prometheus/v1/api.go
// This subset allows us to implement in our tests only the functions we use,
// while allowing compatibility with Prometheus API interface.
type API interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...apiv1.Option) (model.Value, apiv1.Warnings, apiv1.Infos, error)
	QueryRange(ctx context.Context, query string, r apiv1.Range, opts ...apiv1.Option) (model.Value, apiv1.Warnings, apiv1.Infos, error)
}
