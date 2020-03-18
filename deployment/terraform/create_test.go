// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnmarshal validates that we are able to unmarshal
// json output even when TF_LOG variables are set.
func TestUnmarshal(t *testing.T) {
	tf := &Terraform{}
	str := `{
  "dbEndpoint": {
    "sensitive": false,
    "type": "string",
    "value": "dbval"
  },
  "instanceIPs": {
    "sensitive": false,
    "type": [
      "tuple",
      [
        "string"
      ]
    ],
    "value": [
      "1.1.1.1"
    ]
  }
}
2020/03/17 20:14:07 [WARN] Log levels other than TRACE are currently unreliable, and are supported only for backward compatibility.
  Use TF_LOG=TRACE to see Terraform's internal logs.
  ----

`
	output, err := tf.parseOutputJSON([]byte(str))
	require.Nil(t, err)
	assert.Equal(t, output.DBEndpoint.Value, "dbval")
	assert.Equal(t, output.InstanceIps.Value[0], "1.1.1.1")
}
