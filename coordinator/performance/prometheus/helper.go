// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	prometheus "github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	requestTimeout = 60 * time.Second
)

type Helper struct {
	api API
}

// NewHelper creates a helper with the standard Prometheus client
// and API inside it, encapsulating all Prometheus dependencies.
func NewHelper(prometheusURL string) (*Helper, error) {
	config := prometheus.Config{Address: prometheusURL}
	client, err := prometheus.NewClient(config)
	if err != nil {
		return nil, err
	}

	api := apiv1.NewAPI(client)
	helper := &Helper{api}

	return helper, nil
}

// VectorFirst returns the first element from a vector query.
func (p *Helper) VectorFirst(query string) (float64, error) {
	context, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// TODO: use QueryRange to reduce the timespan to query
	value, _, err := p.api.Query(context, query, time.Now())
	if err != nil {
		return 0, err
	}

	return p.extractNumericValueFromFirstElement(value)
}

// Matrix returns the matrix of metrics of a query in the given duration.
// It ensures that the returned matrix has atleast one row.
func (p *Helper) Matrix(query string, startTime, endTime time.Time) (model.Matrix, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// interval 5 seconds
	r := apiv1.Range{
		Start: startTime,
		End:   endTime,
		Step:  5 * time.Second,
	}
	value, _, err := p.api.QueryRange(ctx, query, r)
	if err != nil {
		return nil, err
	}
	if value.Type() != model.ValMatrix {
		return nil, fmt.Errorf("expected a matrix, got a %s", value.Type())
	}

	mat := value.(model.Matrix)
	if len(mat) == 0 {
		return nil, errors.New("matrix has length = 0")
	}

	return mat, nil
}

// SetAPI is used to replace the API interface used to communicate with Prometheus.
// Helpful to mock tests from other packages.
func (p *Helper) SetAPI(api API) {
	p.api = api
}

func (p *Helper) extractNumericValueFromFirstElement(value model.Value) (float64, error) {
	if value.Type() != model.ValVector {
		return 0, fmt.Errorf("expected a vector, got a %s", value.Type().String())
	}

	vec, _ := value.(model.Vector)
	if len(vec) == 0 {
		return 0, errors.New("vector has length = 0")
	}

	textValue := vec[0].Value.String()
	numericValue, err := strconv.ParseFloat(textValue, 64)
	if err != nil {
		return 0, err
	}

	return numericValue, nil
}
