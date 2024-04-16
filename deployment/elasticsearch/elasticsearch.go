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
	"github.com/mattermost/mattermost-load-test-ng/deployment"
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
	creds, err := deployment.GetAWSCreds(awsProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials")
	}

	transport, err := newElasticsearchRoundTripper(sshc, creds, awsRegion)
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

type repositoryResponse struct {
	Type string `json:"type"`
}

// Repository represents a snapshot repository, with a Name and a Type
// (for now only S3 is supported)
type Repository struct {
	Name string
	Type string
}

// ListRepositories returns the list of repositories registered in the server
func (c *Client) ListRepositories() ([]Repository, error) {
	req := esapi.SnapshotGetRepositoryRequest{}
	repositoriesResponse := make(map[string]repositoryResponse)
	if err := c.get(req, &repositoriesResponse); err != nil {
		return nil, fmt.Errorf("unable to perform ListRepositories request: %w", err)
	}

	repositories := []Repository{}
	for k, r := range repositoriesResponse {
		repo := Repository{
			Name: k,
			Type: r.Type,
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// RegisterS3Repository registers a repository of type S3, identified by
// its name, using the role ARN provided
func (c *Client) RegisterS3Repository(name, arn string) error {
	type settings struct {
		Bucket  string `json:"bucket"`
		Region  string `json:"region"`
		RoleARN string `json:"role_arn"`
	}

	type payload struct {
		Type     string   `json:"type"`
		Settings settings `json:"settings"`
	}

	p := payload{
		Type: "s3",
		Settings: settings{
			Bucket:  name,
			Region:  c.awsRegion,
			RoleARN: arn,
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(p); err != nil {
		return fmt.Errorf("unable to encode payload: %w", err)
	}

	req := esapi.SnapshotCreateRepositoryRequest{
		Body:       &buf,
		Repository: name,
	}
	res, err := req.Do(context.Background(), c.client)
	if err != nil {
		return fmt.Errorf("unable to perform RegisterRepository request: %w", err)
	}
	if res.Body == nil {
		return fmt.Errorf("no body returned by RegisterRepository")
	}
	defer res.Body.Close()
	if res.IsError() {
		// Consume body, docs say it's important to do so even if not needed
		io.Copy(io.Discard, res.Body)
		return fmt.Errorf("unable to register repository %q with ARN %q: %q", name, arn, res.String())
	}

	// Consume body, docs say it's important to do so even if not needed
	_, err = io.Copy(io.Discard, res.Body)
	return err
}

type shardsResponse struct {
	Total int `json:"total"`
}

type snapshotResponse struct {
	Name    string         `json:"snapshot"`
	Indices []string       `json:"indices"`
	Shards  shardsResponse `json:"shards"`
}

type snapshotsResponse struct {
	Snapshots []snapshotResponse `json:"snapshots"`
}

// Snapshot represents a snapshot in one of the repositories registered in the
// server, identified by its name, specifying the list of indices within it and
// the total number of shards shared among them.
type Snapshot struct {
	Name        string
	Indices     []string
	TotalShards int
}

// ListSnapshots returns the list of snapshots included in the repository
// specified by repositoryName
func (c *Client) ListSnapshots(repositoryName string) ([]Snapshot, error) {
	req := esapi.SnapshotGetRequest{
		Repository: repositoryName,
		// The API should support listing all snapshots in a repository in
		// *any* other way, but you need to abuse the Snapshot field, which is
		// supposed to be a list of snapshots, to pass a single field that is
		// "_all". Welp.
		Snapshot: []string{"_all"},
	}
	var snapshotsResponse snapshotsResponse
	if err := c.get(req, &snapshotsResponse); err != nil {
		return nil, fmt.Errorf("unable to perform ListSnapshots request: %w", err)
	}

	snapshots := []Snapshot{}
	for _, s := range snapshotsResponse.Snapshots {
		snapshot := Snapshot{
			Name:        s.Name,
			Indices:     s.Indices,
			TotalShards: s.Shards.Total,
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// CloseIndices close the indices specified. This is used so that a snapshot
// containing indices with the same names as the ones provided can be restored;
// otherwise, the restored indices would need to be renamed.
func (c *Client) CloseIndices(indices []string) error {
	req := esapi.IndicesCloseRequest{Index: indices}
	res, err := req.Do(context.Background(), c.client)
	if err != nil {
		return fmt.Errorf("unable to perform CloseIndices request: %w", err)
	}
	if res.Body == nil {
		return fmt.Errorf("no body returned by CloseIndices")
	}
	defer res.Body.Close()
	if res.IsError() {
		// Consume body, docs say it's important to do so even if not needed
		io.Copy(io.Discard, res.Body)
		return fmt.Errorf("unable to close indices %v: %q", indices, res.String())
	}

	// Consume body, docs say it's important to do so even if not needed
	_, err = io.Copy(io.Discard, res.Body)
	return err
}

// RestoreSnapshotOpts exposes two options for configuring the RestoreSnapshot
// request:
//   - WithIndices, a list of the indices from the snapshot that need to be
//     restored.
//   - WithoutIndices, a list of the indices from the snapshot that need to be
//     skipped.
type RestoreSnapshotOpts struct {
	WithIndices    []string
	WithoutIndices []string
}

// MarshalJSON implements the Marshaler interface so that RestoreSnapshotOpts
// can be easily marshalled.
func (r RestoreSnapshotOpts) MarshalJSON() ([]byte, error) {
	indices := []string{}
	indices = append(indices, r.WithIndices...)
	for _, i := range r.WithoutIndices {
		indices = append(indices, "-"+i)
	}

	payload := struct {
		Indices string `json:"indices"`
	}{strings.Join(indices, ",")}

	return json.Marshal(payload)
}

// RestoreSnapshot restores a snapshot from a repository using the options
// provided.
func (c *Client) RestoreSnapshot(repositoryName, snapshotName string, opts RestoreSnapshotOpts) error {
	body, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	req := esapi.SnapshotRestoreRequest{
		Repository: repositoryName,
		Snapshot:   snapshotName,
		Body:       bytes.NewBuffer(body),
	}
	res, err := req.Do(context.Background(), c.client)
	if err != nil {
		return fmt.Errorf("unable to perform RestoreSnapshot request: %w", err)
	}
	if res.Body == nil {
		return fmt.Errorf("no body returned by RestoreSnapshot")
	}
	defer res.Body.Close()
	if res.IsError() {
		// Consume body, docs say it's important to do so even if not needed
		io.Copy(io.Discard, res.Body)
		return fmt.Errorf("unable to restore snapshot %q from repo %q with opts %+v: %q", snapshotName, repositoryName, opts, res.String())
	}

	// Consume body, docs say it's important to do so even if not needed
	_, err = io.Copy(io.Discard, res.Body)
	return err
}

// ListIndices returns the names of the indices already present in the server
// as a plain slice of strings
func (c *Client) ListIndices() ([]string, error) {
	req := esapi.IndicesGetRequest{Index: []string{"_all"}}
	resJSON := make(map[string]json.RawMessage)
	if err := c.get(req, &resJSON); err != nil {
		return nil, fmt.Errorf("unable to perform ListRepositories request: %w", err)
	}

	indices := []string{}
	for index := range resJSON {
		indices = append(indices, index)
	}

	return indices, nil
}

type withPercent struct {
	Percent string `json:"percent"`
}

type indexResponse struct {
	Size  withPercent `json:"size"`
	Files withPercent `json:"files"`
}

type sourceResponse struct {
	Index string `json:"index"`
}

type shardResponse struct {
	Type   string         `json:"type"`
	Stage  string         `json:"stage"`
	Index  indexResponse  `json:"index"`
	Source sourceResponse `json:"source"`
}

type indexRecoveryResponse struct {
	Shards []shardResponse `json:"shards"`
}

type indicesRecoveryResponse map[string]indexRecoveryResponse

// IndexShardRecovery represents the information returned by the IndicesRecovery
// request for a single index shard, specifying:
//   - Index: the name of the index where this shard lives.
//   - Type: the type of the shard, normally "SNAPSHOT" for shards that are
//     being restored from a snapshot.
//   - Stage: the current status of the shard. For SNAPSHOT shards, this is
//     either "INDEX", meaning the shard is still being restored, or "DONE",
//     meaning the shard has already been restored.
//   - Percent: the percentage of bytes already restored. Note this is not the
//     percentage of files already restored.
type IndexShardRecovery struct {
	Index   string
	Type    string
	Stage   string
	Percent string
}

// IndicesRecovery returns status information for each index shard in the server.
// This is useful to track the completion of a snapshot restoration process.
func (c *Client) IndicesRecovery(indices []string) ([]IndexShardRecovery, error) {
	req := esapi.IndicesRecoveryRequest{
		Index: indices,
	}
	indicesRecovery := make(indicesRecoveryResponse)
	if err := c.get(req, &indicesRecovery); err != nil {
		return nil, fmt.Errorf("unable to perform IndicesRecovery request: %w", err)
	}

	recovery := []IndexShardRecovery{}
	for _, resp := range indicesRecovery {
		for _, shard := range resp.Shards {
			recovery = append(recovery, IndexShardRecovery{
				Index: shard.Source.Index,
				Type:  shard.Type,
				Stage: shard.Stage,
				// We're using the percentage of bytes restored,
				// not the percentage of files restored
				Percent: shard.Index.Size.Percent,
			})
		}
	}

	return recovery, nil
}

// requestDoer models all esapi.XYZRequest, which contains a Do function to
// perform the request with the provided client
type requestDoer interface {
	Do(context.Context, esapi.Transport) (*esapi.Response, error)
}

// get runs req.Do, performs the needed checks on the response, and stores the
// result in the value pointed to by result
func (c *Client) get(req requestDoer, result any) error {
	res, err := req.Do(context.Background(), c.client)
	if err != nil {
		return fmt.Errorf("unable to perform request: %w", err)
	}
	if res.Body == nil {
		return fmt.Errorf("no body returned")
	}
	defer res.Body.Close()
	if res.IsError() {
		// Consume body, docs say it's important to do so even if not needed
		io.Copy(io.Discard, res.Body)
		return fmt.Errorf("request failed: %q", res.String())
	}

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(resBytes, result); err != nil {
		return fmt.Errorf("unable to unmarshal response: %w", err)
	}

	return nil
}
