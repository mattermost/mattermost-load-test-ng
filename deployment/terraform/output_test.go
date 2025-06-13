package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstanceConnectionMethods(t *testing.T) {
	instance := Instance{
		PrivateIP:  "10.0.0.1",
		PublicIP:   "203.0.113.1",
		PrivateDNS: "ip-10-0-0-1.internal",
		PublicDNS:  "ec2-203-0-113-1.compute-1.amazonaws.com",
	}
	testCases := []struct {
		name           string
		instance       Instance
		connectionType string
		expectedType   string
		expectedIP     string
		expectedDNS    string
	}{
		{
			name:           "public connection type",
			instance:       instance,
			connectionType: "public",
			expectedType:   "public",
			expectedIP:     instance.PublicIP,
			expectedDNS:    instance.PublicDNS,
		},
		{
			name:           "private connection type",
			instance:       instance,
			connectionType: "private",
			expectedType:   "private",
			expectedIP:     instance.PrivateIP,
			expectedDNS:    instance.PrivateDNS,
		},
		{
			name:           "invalid connection type defaults to public",
			instance:       instance,
			connectionType: "invalid",
			expectedType:   "public",
			expectedIP:     instance.PublicIP,
			expectedDNS:    instance.PublicDNS,
		},
		{
			name:           "empty connection type defaults to public",
			instance:       instance,
			connectionType: "",
			expectedType:   "public",
			expectedIP:     instance.PublicIP,
			expectedDNS:    instance.PublicDNS,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.instance.SetConnectionType(tc.connectionType)

			require.Equal(t, tc.expectedType, tc.instance.GetConnectionType())
			require.Equal(t, tc.expectedIP, tc.instance.GetConnectionIP())
			require.Equal(t, tc.expectedDNS, tc.instance.GetConnectionDNS())
		})
	}
}

func TestSetConnectionType(t *testing.T) {
	t.Run("set public connection type", func(t *testing.T) {
		var i Instance
		i.SetConnectionType("public")
		require.Equal(t, "public", i.GetConnectionType())
	})

	t.Run("set private connection type", func(t *testing.T) {
		var i Instance
		i.SetConnectionType("private")
		require.Equal(t, "private", i.GetConnectionType())
	})

	t.Run("set invalid connection type", func(t *testing.T) {
		var i Instance
		i.SetConnectionType("invalid")
		require.Equal(t, "public", i.GetConnectionType())
	})
}
