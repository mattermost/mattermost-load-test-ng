package elasticsearch

import (
	"net/http"
	"testing"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/stretchr/testify/require"
)

func setupRoundTripper(t *testing.T) *elasticsearchRoundTripper {
	t.Helper()

	extAgent, err := ssh.NewAgent()
	require.NoError(t, err)
	sshc, err := extAgent.NewClient("ip")

	signer := v4.NewSigner()

	// Use the default transport, except for DialContext, for which we use the
	// SSH client dial, effectively tunneling all requests through the SSH
	// connection
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = sshc.DialContextF()

	return &elasticsearchRoundTripper{
		signer:    signer,
		creds:     creds,
		region:    awsRegion,
		transport: transport,
	}, nil
}

func TestSignRequest(t *testing.T) {

}
