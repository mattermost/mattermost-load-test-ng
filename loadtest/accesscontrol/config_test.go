// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package accesscontrol

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() Config {
	return Config{
		Enable:    true,
		UserAgent: "Mattermost/5.13.0 Electron/36.4.0",
		UserAttributes: []UserAttribute{
			{
				Name: "lt_clearance",
				Type: AttributeTypeSelect,
				Values: []AttributeValue{
					{Value: "high", Weight: 0.5},
					{Value: "medium", Weight: 0.25},
					{Value: "low", Weight: 0.25},
				},
			},
			{
				Name: "lt_team",
				Type: AttributeTypeText,
				Values: []AttributeValue{
					{Value: "a", Weight: 0.5},
					{Value: "b", Weight: 0.5},
				},
			},
		},
		SessionAttributes: []SessionAttribute{
			{
				Name:       "vpn_active",
				TTLSeconds: 300,
				Values: []AttributeValue{
					{Value: "true", Weight: 0.6},
					{Value: "false", Weight: 0.4},
				},
			},
		},
		Policies: []Policy{
			{
				Name:                   "read",
				Action:                 ActionChannelRead,
				UserAttribute:          "lt_clearance",
				UserAttributeValues:    []string{"high", "medium"},
				Operator:               OperatorOr,
				SessionAttribute:       "vpn_active",
				SessionAttributeValues: []string{"true"},
			},
			{
				Name:                   "write",
				Action:                 ActionChannelWrite,
				UserAttribute:          "lt_team",
				UserAttributeValues:    []string{"a"},
				SessionAttribute:       "vpn_active",
				SessionAttributeValues: []string{"true", "false"},
			},
		},
	}
}

func TestConfigIsValid(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		require.NoError(t, Config{}.IsValid())
	})

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, testConfig().IsValid())
	})

	for name, tc := range map[string]struct {
		modify func(c *Config)
		err    string
	}{
		"invalid attribute name": {
			modify: func(c *Config) { c.UserAttributes[0].Name = "lt-clearance" },
			err:    "invalid user attribute name",
		},
		"weights not summing to 1": {
			modify: func(c *Config) { c.UserAttributes[0].Values[0].Weight = 0.4 },
			err:    "weights should sum to 1",
		},
		"server derived session attribute": {
			modify: func(c *Config) { c.SessionAttributes[0].Name = "ip_address" },
			err:    "not a session attribute provided by clients",
		},
		"invalid session attribute value": {
			modify: func(c *Config) { c.SessionAttributes[0].Values[0].Value = "yes" },
			err:    `invalid value "yes"`,
		},
		"no policies": {
			modify: func(c *Config) { c.Policies = nil },
			err:    "at least one policy is required",
		},
		"too many policies": {
			modify: func(c *Config) {
				for range maxPermissionPolicies {
					p := c.Policies[0]
					p.Name += "x"
					c.Policies = append(c.Policies, p)
				}
			},
			err: "at most 10 policies",
		},
		"undefined user attribute": {
			modify: func(c *Config) { c.Policies[0].UserAttribute = "unknown" },
			err:    `undefined user attribute "unknown"`,
		},
		"undefined session attribute": {
			modify: func(c *Config) { c.Policies[0].SessionAttribute = "mdm_enrolled" },
			err:    `undefined session attribute "mdm_enrolled"`,
		},
		"policy value not an attribute value": {
			modify: func(c *Config) { c.Policies[0].UserAttributeValues = []string{"top"} },
			err:    `value "top" is not one of the attribute values`,
		},
		"invalid operator": {
			modify: func(c *Config) { c.Policies[0].Operator = "xor" },
			err:    "invalid operator",
		},
		"invalid role": {
			modify: func(c *Config) { c.Policies[0].Role = "channel_user" },
			err:    "invalid role",
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig()
			tc.modify(&cfg)
			require.ErrorContains(t, cfg.IsValid(), tc.err)
		})
	}
}

func TestPolicyExpression(t *testing.T) {
	cfg := testConfig()
	assert.Equal(t, `user.attributes.lt_clearance in ["high", "medium"] || user.session.vpn_active in ["true"]`, cfg.Policies[0].Expression())
	assert.Equal(t, `user.attributes.lt_team in ["a"] && user.session.vpn_active in ["true", "false"]`, cfg.Policies[1].Expression())
}

func TestExpectedPermitRates(t *testing.T) {
	t.Run("exact", func(t *testing.T) {
		rates := testConfig().ExpectedPermitRates()
		// Denied reads: low clearance without VPN.
		assert.InDelta(t, 1-0.25*0.4, rates.Read, 1e-9)
		// Writes also need read access.
		assert.InDelta(t, (1-0.25*0.4)*0.5, rates.Write, 1e-9)
	})

	t.Run("no write policies", func(t *testing.T) {
		cfg := testConfig()
		cfg.Policies = cfg.Policies[:1]
		rates := cfg.ExpectedPermitRates()
		assert.InDelta(t, rates.Read, rates.Write, 1e-9)
	})

	t.Run("non system_user policies are ignored", func(t *testing.T) {
		cfg := testConfig()
		cfg.Policies[1].Role = RoleSystemAdmin
		rates := cfg.ExpectedPermitRates()
		assert.InDelta(t, rates.Read, rates.Write, 1e-9)
	})

	t.Run("sampled", func(t *testing.T) {
		cfg := testConfig()
		// Enough referenced attributes to go over the exact computation
		// limit. The extra policies always grant access.
		for i := range 25 {
			name := "extra_" + strconv.Itoa(i)
			cfg.UserAttributes = append(cfg.UserAttributes, UserAttribute{
				Name:   name,
				Values: []AttributeValue{{Value: "x", Weight: 0.5}, {Value: "y", Weight: 0.5}},
			})
			cfg.Policies = append(cfg.Policies, Policy{
				Name:                   name,
				Action:                 ActionChannelRead,
				UserAttribute:          name,
				UserAttributeValues:    []string{"x", "y"},
				SessionAttribute:       "vpn_active",
				SessionAttributeValues: []string{"true", "false"},
			})
		}
		rates := cfg.ExpectedPermitRates()
		assert.InDelta(t, 1-0.25*0.4, rates.Read, 0.01)
		assert.InDelta(t, (1-0.25*0.4)*0.5, rates.Write, 0.01)
	})
}

func TestPickValue(t *testing.T) {
	values := []AttributeValue{
		{Value: "high", Weight: 0.5},
		{Value: "medium", Weight: 0.25},
		{Value: "low", Weight: 0.25},
	}

	t.Run("deterministic", func(t *testing.T) {
		assert.Equal(t, pickValue("user:a@example.com:x", values), pickValue("user:a@example.com:x", values))
	})

	t.Run("follows the weights", func(t *testing.T) {
		counts := map[string]int{}
		n := 20000
		for i := range n {
			counts[pickValue("key-"+strconv.Itoa(i), values)]++
		}
		for _, v := range values {
			assert.InDelta(t, v.Weight, float64(counts[v.Value])/float64(n), 0.02, v.Value)
		}
	})
}

func TestHeaders(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		assert.Nil(t, Config{}.Headers("user@example.com"))
	})

	t.Run("enabled", func(t *testing.T) {
		cfg := testConfig()
		headers := cfg.Headers("User@Example.com")
		assert.Equal(t, cfg.UserAgent, headers["User-Agent"])

		data, err := base64.StdEncoding.DecodeString(headers[HeaderSessionAttributes])
		require.NoError(t, err)
		var attrs map[string]string
		require.NoError(t, json.Unmarshal(data, &attrs))
		assert.Equal(t, cfg.SessionAttributeValues("user@example.com"), attrs)
	})
}

func TestPolicyID(t *testing.T) {
	id := policyID("lt-read")
	assert.Len(t, id, 26)
	assert.Regexp(t, "^lt[0-9a-f]{24}$", id)
	assert.Equal(t, id, policyID("lt-read"))
	assert.NotEqual(t, id, policyID("lt-write"))
}

func TestSampleConfig(t *testing.T) {
	data, err := os.ReadFile("../../config/config.sample.json")
	require.NoError(t, err)
	var sample struct {
		AccessControlConfiguration Config
	}
	require.NoError(t, json.Unmarshal(data, &sample))

	cfg := sample.AccessControlConfiguration
	require.False(t, cfg.Enable)
	cfg.Enable = true
	require.NoError(t, defaults.Validate(cfg))

	rates := cfg.ExpectedPermitRates()
	assert.InDelta(t, 0.9, rates.Read, 1e-9)
	assert.InDelta(t, 0.9, rates.Write, 1e-9)
}
