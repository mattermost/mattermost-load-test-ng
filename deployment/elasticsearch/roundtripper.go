package elasticsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
)

// elasticsearchRoundTripper implements RoundTrip to use it as a Transport in
// an http client. It signs the requests using AWS Signature Version 4 before
// letting the request go through its inner transport's RoundTrip.
type elasticsearchRoundTripper struct {
	signer    *v4.Signer
	creds     aws.Credentials
	region    string
	transport http.RoundTripper
}

func newElasticsearchRoundTripper(sshc *ssh.Client, creds aws.Credentials, awsRegion string) (*elasticsearchRoundTripper, error) {
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

// RoundTrip implements the RoundTripper interface, signing the request with AWS
// Signature Version 4 before passing it to the underlying transport's RoundTrip
func (s elasticsearchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := s.signRequest(req); err != nil {
		return nil, err
	}
	return s.transport.RoundTrip(req)
}

// signRequest sign the provided request using AWS Signature Version 4
func (s elasticsearchRoundTripper) signRequest(req *http.Request) error {
	body := []byte{}
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			return fmt.Errorf("unable to read request's body: %w", err)
		}

		// Restore the request's body so that it can be read again
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	bodySha := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(bodySha[:])

	return s.signer.SignHTTP(context.Background(), s.creds, req, payloadHash, "es", s.region, time.Now())
}
