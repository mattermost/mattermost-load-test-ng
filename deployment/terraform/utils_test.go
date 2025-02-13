// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"os"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestFillConfigTemplate(t *testing.T) {
	t.Run("empty template", func(t *testing.T) {
		input := make(map[string]any)
		output, err := fillConfigTemplate("", input)
		require.NoError(t, err)
		require.Empty(t, output)
	})

	t.Run("no data", func(t *testing.T) {
		tmpl := "template"
		output, err := fillConfigTemplate(tmpl, nil)
		require.NoError(t, err)
		require.Equal(t, tmpl, output)
	})

	t.Run("valid data", func(t *testing.T) {
		tmpl := "this is a {{.value}}"
		data := map[string]any{"value": "template"}
		output, err := fillConfigTemplate(tmpl, data)
		require.NoError(t, err)
		require.Equal(t, "this is a template", output)
	})
}

func TestGetServerURL(t *testing.T) {
	for _, tc := range []struct {
		name     string
		output   *Output
		config   *deployment.Config
		expected string
	}{
		{
			name: "no proxy, no siteurl",
			output: &Output{
				Instances: []Instance{{
					PublicIP: "localhost",
				}},
			},
			config:   &deployment.Config{},
			expected: "localhost:8065",
		}, {
			name: "proxy, no siteurl",
			output: &Output{
				Instances: []Instance{{
					PrivateIP: "localhost",
				}},
				Proxies: []Instance{{
					PrivateIP: "proxy_ip",
				}},
			},
			config:   &deployment.Config{},
			expected: "proxy_ip",
		}, {
			name: "no proxy, siteurl",
			output: &Output{
				Instances: []Instance{{
					PrivateIP: "localhost",
				}},
			},
			config: &deployment.Config{
				SiteURL: "ltserver",
			},
			expected: "ltserver:8065",
		}, {
			name: "proxy, siteurl",
			output: &Output{
				Instances: []Instance{{
					PrivateIP: "localhost",
				}},
				Proxies: []Instance{{
					PrivateIP: "proxy_ip",
				}},
			},
			config: &deployment.Config{
				SiteURL: "ltserver",
			},
			expected: "ltserver",
		}, {
			name: "serverurl takes priority",
			output: &Output{
				Instances: []Instance{{
					PrivateIP: "localhost",
				}},
				Proxies: []Instance{{
					PrivateIP: "proxy_ip",
				}},
			},
			config: &deployment.Config{
				SiteURL:   "siteurl",
				ServerURL: "serverurl",
			},
			expected: "serverurl",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, getServerURL(tc.output, tc.config))
		})
	}
}

func TestValidateLicense(t *testing.T) {
	testLicensePath := "./testdata/testlicense.mattermost-license"

	t.Run("valid license with matching service environment", func(t *testing.T) {
		oldValue := os.Getenv("MM_SERVICEENVIRONMENT")
		defer func() { os.Setenv("MM_SERVICEENVIRONMENT", oldValue) }()

		os.Setenv("MM_SERVICEENVIRONMENT", model.ServiceEnvironmentTest)
		err := validateLicense(testLicensePath)
		require.NoError(t, err)
	})

	t.Run("valid license with different service environment", func(t *testing.T) {
		oldValue := os.Getenv("MM_SERVICEENVIRONMENT")
		defer func() { os.Setenv("MM_SERVICEENVIRONMENT", oldValue) }()

		os.Setenv("MM_SERVICEENVIRONMENT", model.ServiceEnvironmentProduction)
		err := validateLicense(testLicensePath)
		require.EqualError(t, err, "this license is valid only with a \"test\" service environment, which is currently set to \"production\"; try adding the -service_environment=test flag to change it")
	})

	t.Run("invalid license", func(t *testing.T) {
		// Create a temp license file
		file, err := os.CreateTemp(os.TempDir(), "license")
		require.NoError(t, err)

		// Fill it with random data and close it
		randomData := []byte{1, 2, 3}
		_, err = file.Write(randomData)
		require.NoError(t, err)
		err = file.Close()
		require.NoError(t, err)

		err = validateLicense(file.Name())
		require.ErrorContains(t, err, "failed to validate license:")
	})
}
