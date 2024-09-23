// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/stretchr/testify/require"
)

func TestFillConfigTemplate(t *testing.T) {
	t.Run("empty template", func(t *testing.T) {
		input := make(map[string]string)
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
		data := map[string]string{"value": "template"}
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
				Proxy: Instance{
					PrivateIP: "proxy_ip",
				},
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
				Proxy: Instance{
					PrivateIP: "proxy_ip",
				},
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
				Proxy: Instance{
					PrivateIP: "proxy_ip",
				},
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
