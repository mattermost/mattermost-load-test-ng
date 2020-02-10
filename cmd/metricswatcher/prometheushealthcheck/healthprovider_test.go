// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prometheushealthcheck

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"
)

func Test_CanCreateAHealthProvider(t *testing.T) {
	healthProvider, err := NewHealthProvider("prometheus:9090")

	assert.NotNil(t, healthProvider)
	assert.Nil(t, err)
}

func Test_HealthProvider_ReturnsTrueWhenHealthIsUp(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("Prometheus is Healthy.")); err != nil {
			assert.Fail(t, err.Error())
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	healthProvider, _ := NewHealthProvider(server.URL)
	result := healthProvider.Check()

	assert.True(t, result.Healthy)
	assert.Nil(t, result.Error)
	assert.False(t, result.Timestamp.IsZero())
}

func Test_HealthProvider_ReturnsFalseWhenHealthIsDown(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)

		if _, err := w.Write([]byte("Prometheus is a Teapot.")); err != nil {
			assert.Fail(t, err.Error())
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	healthProvider, _ := NewHealthProvider(server.URL)
	result := healthProvider.Check()

	assert.False(t, result.Healthy)
	assert.Equal(t, "Prometheus is a Teapot.", result.Error.Error())
	assert.False(t, result.Timestamp.IsZero())
}
