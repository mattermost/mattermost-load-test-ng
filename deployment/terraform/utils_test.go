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
					PrivateIP: "localhost",
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
	testLicenseData := []byte("eyJpZCI6Iks5aEdwYkhlaHFiNUY0S3BQM3phb05xWjNMIiwiaXNzdWVkX2F0IjoxNzI4OTkyMzg4NTE1LCJzdGFydHNfYXQiOjE3Mjg5OTIzODg1MTUsImV4cGlyZXNfYXQiOjE3OTIwMzY4MDAwMDAsInNrdV9uYW1lIjoiRW50ZXJwcmlzZSIsInNrdV9zaG9ydF9uYW1lIjoiZW50ZXJwcmlzZSIsImN1c3RvbWVyIjp7ImlkIjoicDl1bjM2OWE2N2tzbWo0eWQ2aTZpYjM5d2giLCJuYW1lIjoiRGV2IExvYWQgVGVzdCIsImVtYWlsIjoibWFyaWEubnVuZXpAbWF0dGVybW9zdC5jb20iLCJjb21wYW55IjoiRGV2IExvYWQgVGVzdCJ9LCJmZWF0dXJlcyI6eyJ1c2VycyI6MSwibGRhcCI6dHJ1ZSwibGRhcF9ncm91cHMiOnRydWUsIm1mYSI6dHJ1ZSwiZ29vZ2xlX29hdXRoIjp0cnVlLCJvZmZpY2UzNjVfb2F1dGgiOnRydWUsImNvbXBsaWFuY2UiOnRydWUsImNsdXN0ZXIiOnRydWUsIm1ldHJpY3MiOnRydWUsIm1ocG5zIjp0cnVlLCJzYW1sIjp0cnVlLCJlbGFzdGljX3NlYXJjaCI6dHJ1ZSwiYW5ub3VuY2VtZW50Ijp0cnVlLCJ0aGVtZV9tYW5hZ2VtZW50Ijp0cnVlLCJlbWFpbF9ub3RpZmljYXRpb25fY29udGVudHMiOnRydWUsImRhdGFfcmV0ZW50aW9uIjp0cnVlLCJtZXNzYWdlX2V4cG9ydCI6dHJ1ZSwiY3VzdG9tX3Blcm1pc3Npb25zX3NjaGVtZXMiOnRydWUsImN1c3RvbV90ZXJtc19vZl9zZXJ2aWNlIjp0cnVlLCJndWVzdF9hY2NvdW50cyI6dHJ1ZSwiZ3Vlc3RfYWNjb3VudHNfcGVybWlzc2lvbnMiOnRydWUsImlkX2xvYWRlZCI6dHJ1ZSwibG9ja190ZWFtbWF0ZV9uYW1lX2Rpc3BsYXkiOnRydWUsImNsb3VkIjpmYWxzZSwic2hhcmVkX2NoYW5uZWxzIjp0cnVlLCJyZW1vdGVfY2x1c3Rlcl9zZXJ2aWNlIjp0cnVlLCJvcGVuaWQiOnRydWUsImVudGVycHJpc2VfcGx1Z2lucyI6dHJ1ZSwiYWR2YW5jZWRfbG9nZ2luZyI6dHJ1ZSwiZnV0dXJlX2ZlYXR1cmVzIjp0cnVlfSwiaXNfdHJpYWwiOmZhbHNlLCJpc19nb3Zfc2t1IjpmYWxzZX228H2ThUFYKzA3c4Zfrp5ETKKG4V9aUAHKenAhFRjH6KV6WtcoW8LA/b6KDVyvaWsCY7yvrh+ZsYrDVF26uKipxvPcy9x1ia6DVDr1nVraR+DzaWy6V4en3qbwiWWVuHhlTjioeLetoGuGnHuLZPLN8IFfpijX3vo78w/A8z603lMzWLaO9yIdV+SqL5/+0SScgvPRIWqG+dZlzrxehkCLZH9peZnxnbmNLkXdbfNbBVK8wLekh3hZqFotXCA9i/btn7CtDlyKIZdIs4z81hCKt4EEFj0CjtJmhlUVhTm8c4D/McylAKKgEbY1FKEVKnmGj4mGlbewfH6D5I3r8W52")

	t.Run("valid license with matching service environment", func(t *testing.T) {
		oldValue := os.Getenv("MM_SERVICEENVIRONMENT")
		defer func() { os.Setenv("MM_SERVICEENVIRONMENT", oldValue) }()

		os.Setenv("MM_SERVICEENVIRONMENT", model.ServiceEnvironmentTest)
		err := validateLicense(testLicenseData)
		require.NoError(t, err)
	})

	t.Run("valid license with different service environment", func(t *testing.T) {
		oldValue := os.Getenv("MM_SERVICEENVIRONMENT")
		defer func() { os.Setenv("MM_SERVICEENVIRONMENT", oldValue) }()

		os.Setenv("MM_SERVICEENVIRONMENT", model.ServiceEnvironmentProduction)
		err := validateLicense(testLicenseData)
		require.EqualError(t, err, "this license is valid only with a \"test\" service environment, which is currently set to \"production\"; try adding the -service_environment=test flag to change it")
	})

	t.Run("invalid license", func(t *testing.T) {
		randomData := []byte{1, 2, 3}
		err := validateLicense(randomData)
		require.ErrorContains(t, err, "failed to validate license:")
	})
}
