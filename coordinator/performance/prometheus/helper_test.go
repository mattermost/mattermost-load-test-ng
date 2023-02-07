// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type testAPI struct {
	value    model.Value
	warnings apiv1.Warnings
	err      error
}

func (t testAPI) Query(ctx context.Context, query string, ts time.Time, opts ...apiv1.Option) (model.Value, apiv1.Warnings, error) {
	return t.value, t.warnings, t.err
}

func (t testAPI) QueryRange(ctx context.Context, query string, r apiv1.Range, opts ...apiv1.Option) (model.Value, apiv1.Warnings, error) {
	return t.value, t.warnings, nil
}

func Test_VectorFirst_ReturnsFloatValue(t *testing.T) {
	expectedValue := float64(12.12345)
	sample := &model.Sample{Value: model.SampleValue(expectedValue)}
	vector := []*model.Sample{sample}

	api := testAPI{
		value: model.Vector(vector),
	}

	helper := Helper{api}
	actualValue, err := helper.VectorFirst("some PromQL query")

	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)
}

func Test_VectorFirst_FailsIfItsNotAVector(t *testing.T) {
	api := testAPI{
		value: &model.String{Value: "oh no!"},
	}

	helper := Helper{api}
	actualValue, err := helper.VectorFirst("some PromQL query")

	assert.Equal(t, "expected a vector, got a string", err.Error())
	assert.Equal(t, float64(0), actualValue)
}
