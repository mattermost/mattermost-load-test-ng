// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureFlagsEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		assert.Equal(t, []string{
			"MM_FEATUREFLAGS_CHANNELBOOKMARKS=true",
			"MM_FEATUREFLAGS_CUSTOMPROFILEATTRIBUTES=true",
			"MM_FEATUREFLAGS_POSTPRIORITY=true",
			"MM_FEATUREFLAGS_WEBSOCKETEVENTSCOPE=true",
		}, featureFlagsEnv(&deployment.Config{}))
	})

	t.Run("access control and overrides", func(t *testing.T) {
		cfg := &deployment.Config{
			AccessControlSettings: deployment.AccessControlSettings{Enable: true},
			MattermostFeatureFlags: map[string]string{
				"PostPriority":      "false",
				"SessionAttributes": "false",
				"SomeNewFlag":       "true",
			},
		}
		assert.Equal(t, []string{
			"MM_FEATUREFLAGS_CHANNELBOOKMARKS=true",
			"MM_FEATUREFLAGS_CUSTOMPROFILEATTRIBUTES=true",
			"MM_FEATUREFLAGS_PERMISSIONPOLICIES=true",
			"MM_FEATUREFLAGS_POSTPRIORITY=false",
			"MM_FEATUREFLAGS_SESSIONATTRIBUTES=false",
			"MM_FEATUREFLAGS_SOMENEWFLAG=true",
			"MM_FEATUREFLAGS_WEBSOCKETEVENTSCOPE=true",
		}, featureFlagsEnv(cfg))
	})
}

func TestServiceFileFeatureFlags(t *testing.T) {
	for name, serviceFile := range map[string]string{
		"app server": mattermostServiceFile,
		"job server": jobServerServiceFile,
	} {
		t.Run(name, func(t *testing.T) {
			tmpl, err := template.New("serviceFile").Parse(serviceFile)
			require.NoError(t, err)

			var out bytes.Buffer
			require.NoError(t, tmpl.Execute(&out, map[string]any{
				"ServiceEnvironment": "test",
				"User":               "ubuntu",
				"FeatureFlags":       []string{"MM_FEATUREFLAGS_A=true", "MM_FEATUREFLAGS_B=false"},
			}))
			assert.Contains(t, out.String(), "LimitNOFILE=49152\nEnvironment=MM_FEATUREFLAGS_A=true\nEnvironment=MM_FEATUREFLAGS_B=false\nEnvironment=MM_SERVICEENVIRONMENT=test\n")
		})
	}
}
