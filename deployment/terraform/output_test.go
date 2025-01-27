package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)


func TestInstanceConnectionMethods(t *testing.T) {
	testCases := []struct {
		name           string
		instance       Instance
		connectionType string
		expectedType   string
		expectedIP     string
		expectedDNS    string
	}{
		{
			name: "public connection type",
			instance: Instance{
				PrivateIP:  "10.0.0.1",
				PublicIP:   "203.0.113.1",
				PrivateDNS: "ip-10-0-0-1.internal",
				PublicDNS:  "ec2-203-0-113-1.compute-1.amazonaws.com",
			},
			connectionType: "public",
			expectedType:   "public",
			expectedIP:     "203.0.113.1",
			expectedDNS:    "ec2-203-0-113-1.compute-1.amazonaws.com",
		},
		{
			name: "private connection type",
			instance: Instance{
				PrivateIP:  "10.0.0.2",
				PublicIP:   "203.0.113.2",
				PrivateDNS: "ip-10-0-0-2.internal",
				PublicDNS:  "ec2-203-0-113-2.compute-1.amazonaws.com",
			},
			connectionType: "private",
			expectedType:   "private",
			expectedIP:     "10.0.0.2",
			expectedDNS:    "ip-10-0-0-2.internal",
		},
		{
			name: "invalid connection type defaults to public",
			instance: Instance{
				PrivateIP:  "10.0.0.3",
				PublicIP:   "203.0.113.3",
				PrivateDNS: "ip-10-0-0-3.internal",
				PublicDNS:  "ec2-203-0-113-3.compute-1.amazonaws.com",
			},
			connectionType: "invalid",
			expectedType:   "public",
			expectedIP:     "203.0.113.3",
			expectedDNS:    "ec2-203-0-113-3.compute-1.amazonaws.com",
		},
		{
			name: "empty connection type defaults to public",
			instance: Instance{
				PrivateIP:  "10.0.0.4",
				PublicIP:   "203.0.113.4",
				PrivateDNS: "ip-10-0-0-4.internal",
				PublicDNS:  "ec2-203-0-113-4.compute-1.amazonaws.com",
			},
			connectionType: "",
			expectedType:   "public",
			expectedIP:     "203.0.113.4",
			expectedDNS:    "ec2-203-0-113-4.compute-1.amazonaws.com",
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
