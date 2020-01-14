package prometheushelper

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

type PrometheusHelper struct {
	API prometheusAPI
}

// NewPrometheusHelper creates a helper with the standard Prometheus client
// and API inside it, encapsulating all Prometheus dependencies.
func NewPrometheusHelper(prometheusURL string) (*PrometheusHelper, error) {
	config := prometheus.Config{Address: prometheusURL}
	client, err := prometheus.NewClient(config)

	if err != nil {
		return nil, err
	}

	api := apiv1.NewAPI(client)
	prometheusHelper := &PrometheusHelper{api}

	return prometheusHelper, nil
}

func (p PrometheusHelper) VectorFirst(query string) (float64, error) {
	context := context.Background()
	ts := time.Now()

	value, _, err := p.API.Query(context, query, ts)

	if err != nil {
		return 0, err
	}

	return p.extractNumericValueFromFirstElement(value)
}

func (p PrometheusHelper) extractNumericValueFromFirstElement(value model.Value) (float64, error) {
	if value.Type() != model.ValVector {
		return 0, fmt.Errorf("Expected a vector, got a %s", value.Type().String())
	}

	vec, _ := value.(model.Vector)

	if len(vec) == 0 {
		return 0, errors.New("Vector has length = 0")
	}

	textValue := vec[0].Value.String()
	numericValue, err := strconv.ParseFloat(textValue, 64)

	if err != nil {
		return 0, err
	}

	return numericValue, nil
}
