package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	es "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
)

// Client is a wrapper on top of the official go-elasticsearch
// client, implementing a custom transport that modifies the request in two
// ways:
//   - Requests are signed using AWS Signature Version 4 before passing them
//     to the underlying transport
//   - The transport is tunneled through the provided SSH client using its Dial
//     function
type Client struct {
	client    *es.Client
	awsRegion string
	transport *elasticsearchRoundTripper
}

// New builds a new Client for the AWS OpenSearch Domain pointed to by
// esEndoint, using the AWS profile to sign requests and tunneling those
// requests through the SSH connection provided by the SSH client.
func New(esEndpoint string, sshc *ssh.Client, awsProfile, awsRegion string) (*Client, error) {
	transport, err := newElasticsearchRoundTripper(sshc, awsProfile, awsRegion)
	if err != nil {
		return nil, err
	}

	client, err := es.NewClient(es.Config{
		Addresses: []string{"https://" + esEndpoint},
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	return &Client{client, awsRegion, transport}, nil
}
