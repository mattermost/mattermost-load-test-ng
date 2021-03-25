// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"testing"

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
