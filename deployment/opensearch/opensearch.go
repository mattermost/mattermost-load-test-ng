package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Client is a wrapper on top of the official opensearch
// client, implementing a custom transport that modifies the request in two
// ways:
//   - Requests are signed using AWS Signature Version 4 before passing them
//     to the underlying transport
//   - The transport is tunneled through the provided SSH client using its Dial
//     function
type Client struct {
	client    *opensearchapi.Client
	awsRegion string
	transport *opensearchRoundTripper
}

// New builds a new Client for the AWS OpenSearch Domain pointed to by
// esEndoint, using the AWS profile to sign requests and tunneling those
// requests through the SSH connection provided by the SSH client.
func New(esEndpoint string, sshc *ssh.Client, awsProfile, awsRegion string) (*Client, error) {
	creds, err := deployment.GetAWSCreds(awsProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials")
	}

	transport, err := newOpensearchRoundTripper(sshc.DialContextF(), creds, awsRegion)
	if err != nil {
		return nil, err
	}

	client, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: opensearch.Config{
			Addresses: []string{"https://" + esEndpoint},
			Transport: transport,
		},
	})

	if err != nil {
		return nil, err
	}

	return &Client{client, awsRegion, transport}, nil
}

// Repository represents a snapshot repository, with a Name and a Type
// (for now only S3 is supported)
type Repository struct {
	Name string
	Type string
}

// ListRepositories returns the list of repositories registered in the server
func (c *Client) ListRepositories() ([]Repository, error) {
	resp, err := c.client.Snapshot.Repository.Get(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to perform ListRepositories request: %w", err)
	}

	repositories := []Repository{}
	for k, r := range resp.Repos {
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

	res, err := c.client.Snapshot.Repository.Create(context.Background(), opensearchapi.SnapshotRepositoryCreateReq{
		Repo: name,
		Body: &buf,
	})
	if err != nil {
		return fmt.Errorf("unable to perform RegisterRepository request: %w", err)
	}
	if res.Inspect().Response.Body == nil {
		return fmt.Errorf("no body returned by RegisterRepository")
	}
	return nil
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
	resp, err := c.client.Snapshot.Get(context.Background(), opensearchapi.SnapshotGetReq{
		Repo: repositoryName,
		// The API should support listing all snapshots in a repository in
		// *any* other way, but you need to abuse the Snapshot field, which is
		// supposed to be a list of snapshots, to pass a single field that is
		// "_all". Welp.
		Snapshots: []string{"_all"},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to perform ListSnapshots request: %w", err)
	}

	snapshots := []Snapshot{}
	for _, s := range resp.Snapshots {
		snapshot := Snapshot{
			Name:        s.Snapshot,
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
	resp, err := c.client.Indices.Close(context.Background(), opensearchapi.IndicesCloseReq{
		Index: strings.Join(indices, ","),
	})
	if err != nil {
		return fmt.Errorf("unable to perform CloseIndices request: %w", err)
	}
	if resp.Inspect().Response.Body == nil {
		return fmt.Errorf("no body returned by CloseIndices")
	}
	return nil
}

// RestoreSnapshotOpts exposes three options for configuring the RestoreSnapshot
// request:
//   - WithIndices, a list of the indices from the snapshot that need to be
//     restored.
//   - WithoutIndices, a list of the indices from the snapshot that need to be
//     skipped.
//   - NumberOfReplicas, the number of replicas each primary shard has.
//     Defaults to 1.
type RestoreSnapshotOpts struct {
	WithIndices      []string
	WithoutIndices   []string
	NumberOfReplicas int
}

// MarshalJSON implements the Marshaler interface so that RestoreSnapshotOpts
// can be easily marshalled.
func (r RestoreSnapshotOpts) MarshalJSON() ([]byte, error) {
	type restoreSnapshotBodyIndexSettings struct {
		NumReplicas int `json:"index.number_of_replicas"`
	}

	type restoreSnapshotBody struct {
		Indices       string                           `json:"indices"`
		IndexSettings restoreSnapshotBodyIndexSettings `json:"index_settings"`
	}

	// Set the indices we want and the ones we want to exclude
	indices := []string{}
	indices = append(indices, r.WithIndices...)
	for _, i := range r.WithoutIndices {
		indices = append(indices, "-"+i)
	}

	payload := restoreSnapshotBody{
		Indices:       strings.Join(indices, ","),
		IndexSettings: restoreSnapshotBodyIndexSettings{r.NumberOfReplicas},
	}

	return json.Marshal(payload)
}

func (r RestoreSnapshotOpts) IsValid() error {
	if r.NumberOfReplicas < 0 {
		return fmt.Errorf("number of replicas must be at least 0, but it is %d", r.NumberOfReplicas)
	}

	return nil
}

// RestoreSnapshot restores a snapshot from a repository using the options
// provided.
func (c *Client) RestoreSnapshot(repositoryName, snapshotName string, opts RestoreSnapshotOpts) error {
	if err := opts.IsValid(); err != nil {
		return fmt.Errorf("invalid options for restoring the snapshot: %w", err)
	}

	body, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	res, err := c.client.Snapshot.Restore(context.Background(), opensearchapi.SnapshotRestoreReq{
		Repo:     repositoryName,
		Snapshot: snapshotName,
		Body:     bytes.NewBuffer(body),
	})
	if err != nil {
		return fmt.Errorf("unable to perform RestoreSnapshot request: %w", err)
	}
	if res.Inspect().Response.Body == nil {
		return fmt.Errorf("no body returned by RestoreSnapshot")
	}
	return nil
}

// ListIndices returns the names of the indices already present in the server
// as a plain slice of strings
func (c *Client) ListIndices() ([]string, error) {
	resp, err := c.client.Indices.Get(context.Background(), opensearchapi.IndicesGetReq{
		Indices: []string{"_all"},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to perform ListRepositories request: %w", err)
	}

	indices := []string{}
	for index := range resp.Indices {
		indices = append(indices, index)
	}

	return indices, nil
}

// SnapshotIndexShardRecovery represents the information returned by the IndicesRecovery
// request for a single index shard, specifying:
//   - Index: the name of the index where this shard lives.
//   - Type: the type of the shard, normally "SNAPSHOT" for shards that are
//     being restored from a snapshot.
//   - Stage: the current status of the shard. For SNAPSHOT shards, this is
//     either "INDEX", meaning the shard is still being restored, or "DONE",
//     meaning the shard has already been restored.
//   - Percent: the percentage of bytes already restored. Note this is not the
//     percentage of files already restored.
type SnapshotIndexShardRecovery struct {
	Index   string
	Stage   string
	Percent string
}

// SnapshotIndicesRecovery returns status information for each index shard in
// the server of type SNAPSHOT.
// This is useful to track the completion of a snapshot restoration process.
func (c *Client) SnapshotIndicesRecovery(indices []string) ([]SnapshotIndexShardRecovery, error) {
	resp, err := c.client.Indices.Recovery(context.Background(), &opensearchapi.IndicesRecoveryReq{
		Indices: indices,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to perform IndicesRecovery request: %w", err)
	}
	recovery := []SnapshotIndexShardRecovery{}
	for _, resp := range resp.Indices {
		for _, shard := range resp.Shards {
			// Add only the shards corresponding to the snapshot restoration
			if shard.Type != "SNAPSHOT" {
				continue
			}
			recovery = append(recovery, SnapshotIndexShardRecovery{
				Index: shard.Source.Name,
				Stage: shard.Stage,
				// We're using the percentage of bytes restored,
				// not the percentage of files restored
				Percent: shard.Index.Size.Percent,
			})
		}
	}

	return recovery, nil
}

const (
	ClusterStatusGreen  = "green"
	ClusterStatusYellow = "yellow"
	ClusterStatusRed    = "red"
)

type ClusterHealthResponse struct {
	Status             string `json:"status"`
	InitializingShards int    `json:"initializing_shards"`
	UnassignedShards   int    `json:"unassigned_shards"`
}

func (c *Client) ClusterHealth() (ClusterHealthResponse, error) {
	resp, err := c.client.Cluster.Health(context.Background(), &opensearchapi.ClusterHealthReq{
		Indices: []string{"_all"},
	})
	if err != nil {
		return ClusterHealthResponse{}, fmt.Errorf("unable to perform ClusterHealth request: %w", err)
	}

	return ClusterHealthResponse{
		Status:             resp.Status,
		InitializingShards: resp.InitializingShards,
		UnassignedShards:   resp.UnassignedShards,
	}, nil
}
