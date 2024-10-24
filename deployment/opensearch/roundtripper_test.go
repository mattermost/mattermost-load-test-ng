package opensearch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gliderlabs/ssh"
	sshserver "github.com/gliderlabs/ssh"
	ltssh "github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/stretchr/testify/require"
)

const (
	accessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	secretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	region          = "us-east-1"
	service         = "es"
	sshIP           = "127.0.0.1"
	sshPort         = ":2222"
)

var awsDummyCreds = aws.Credentials{
	AccessKeyID:     accessKeyID,
	SecretAccessKey: secretAccessKey,
	SessionToken:    "",
	Source:          "Environment",
	CanExpire:       false,
	Expires:         time.Time{},
}

func TestRoundTrip(t *testing.T) {
	setupSSHServer(t)
	sshc := setupSSHClient(t)

	// Custom DialContext function that simply calls sshc's DialContext after
	// setting dialCtxtCalled to true, so that we can check that this function
	// was actually called
	dialCtxtCalled := false
	dialCtxtF := func(ctxt context.Context, network string, addr string) (net.Conn, error) {
		dialCtxtCalled = true
		return sshc.DialContextF()(ctxt, network, addr)
	}

	// Build the roundtripper
	roundtripper, err := newOpensearchRoundTripper(dialCtxtF, awsDummyCreds, region)
	require.NoError(t, err)

	// Build a dummy request with no headers
	req := httptest.NewRequest("GET", "https://example.com", nil)
	require.Empty(t, req.Header)

	// Perform the request
	_, err = roundtripper.RoundTrip(req)
	require.NoError(t, err)

	// Check that the provided DialContext function was called and that the
	// request was signed
	require.True(t, dialCtxtCalled)
	checkSignature(t, req)
}

func setupSSHServer(t *testing.T) {
	t.Helper()
	// Dummy SSH server allowing port forwarding
	server := sshserver.Server{
		Addr: sshIP + sshPort,
		LocalPortForwardingCallback: sshserver.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			return true
		}),
	}

	// Close all active connections and shutdown the server when the test ends
	t.Cleanup(func() {
		err := server.Close()
		require.NoError(t, err)
		err = server.Shutdown(context.Background())
		require.NoError(t, err)
	})

	// Non-blocking ListenAndServe, which will finish as soon as the test is
	// finished and the cleanup function is called
	go func() {
		err := server.ListenAndServe()
		require.Equal(t, sshserver.ErrServerClosed, err)
	}()
}

func setupSSHClient(t *testing.T) *ltssh.Client {
	t.Helper()

	extAgent, err := ltssh.NewAgent()
	require.NoError(t, err)

	// Wait for the SSH server to start
	var sshc *ltssh.Client
	require.Eventually(t, func() bool {
		sshc, err = extAgent.NewClientWithPort(sshIP, sshPort)
		return err == nil
	}, 5*time.Second, 100*time.Millisecond)

	return sshc
}

func checkSignature(t *testing.T, req *http.Request) {
	t.Helper()

	// Check that the Authorization header was correctly added
	require.Contains(t, req.Header, "Authorization")
	authHeader := req.Header["Authorization"]
	require.Len(t, authHeader, 1)
	authStr := authHeader[0]

	// Split the Authorization header, which is of the form:
	// "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240416/us-east-1/es/aws4_request, SignedHeaders=host;x-amz-date, Signature=8c9b21d9b7558a3c4dea875130652aaa7d216affbdad8521e6c1c7376dd7cf51"
	authItems := strings.Split(authStr, " ")
	require.Len(t, authItems, 4)

	// Get the four different items in the string
	algorithm := authItems[0]
	credential := strings.TrimSuffix(authItems[1], ",")
	signedHeaders := strings.TrimSuffix(authItems[2], ",")
	signature := strings.TrimSuffix(authItems[3], ",")

	// Check all items were correctly computed
	require.Equal(t, algorithm, "AWS4-HMAC-SHA256")
	today := time.Now().Format("20060102")
	expectedCredential := fmt.Sprintf("Credential=%s/%s/%s/%s/aws4_request", accessKeyID, today, region, service)
	require.Equal(t, expectedCredential, credential)
	require.Equal(t, "SignedHeaders=host;x-amz-date", signedHeaders)
	require.Len(t, strings.TrimPrefix(signature, "Signature="), 64)
}
