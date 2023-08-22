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
// https://github.com/prometheus/client_golang/blob/release-1.14/api/prometheus/v1/api.go#L221-L266
// This subset allows us to implement in our tests only the functions we use,
// while allowing compatibility with Prometheus API interface.
type API interface {
	Query(ctx context.Context, query string, ts time.Time, opts ...apiv1.Option) (model.Value, apiv1.Warnings, error)
	QueryRange(ctx context.Context, query string, r apiv1.Range, opts ...apiv1.Option) (model.Value, apiv1.Warnings, error)
}
