// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

var esDomainNameRe = regexp.MustCompile(`^[a-z][a-z0-9\-]{2,27}$`)

// Config contains the necessary data
// to deploy and provision a load test environment.
type Config struct {
	// AWSProfile is the name of the AWS profile to use for all AWS commands
	AWSProfile string `default:"mm-loadtest"`
	// AWSRegion is the region used to deploy all resources.
	AWSRegion string `default:"us-east-1"`
	// AWSAMI is the AMI to use for all EC2 instances.
	AWSAMI string `default:"ami-0fa37863afb290840"`
	// ClusterName is the name of the cluster.
	ClusterName string `default:"loadtest" validate:"alpha"`
	// ClusterVpcID is the id of the VPC associated to the resources.
	ClusterVpcID string
	// ClusterSubnetID is the id of the subnet associated to the resources.
	ClusterSubnetID string
	// Number of application instances.
	AppInstanceCount int `default:"1" validate:"range:[0,)"`
	// Type of the EC2 instance for app.
	AppInstanceType string `default:"c7i.xlarge" validate:"notempty"`
	// Number of agents, first agent and coordinator will share the same instance.
	AgentInstanceCount int `default:"2" validate:"range:[0,)"`
	// Type of the EC2 instance for agent.
	AgentInstanceType string `default:"c7i.xlarge" validate:"notempty"`
	// Logs the command output (stdout & stderr) to home directory.
	EnableAgentFullLogs bool `default:"true"`
	// Number of proxy instances.
	ProxyInstanceCount int `default:"1" validate:"range:[0,1]"`
	// Type of the EC2 instance for proxy.
	ProxyInstanceType string `default:"m4.xlarge" validate:"notempty"`
	// Path to the SSH public key.
	SSHPublicKey string `default:"~/.ssh/id_rsa.pub" validate:"notempty"`
	// Terraform database connection and provision settings.
	TerraformDBSettings TerraformDBSettings
	// External database connection settings
	ExternalDBSettings ExternalDBSettings
	// External bucket connection settings.
	ExternalBucketSettings ExternalBucketSettings
	// ExternalAuthProviderSettings contains the settings for configuring an external auth provider.
	ExternalAuthProviderSettings ExternalAuthProviderSettings
	// MattermostDownloadURL supports the following use cases:
	// 1. If it is a URL, it should be the Mattermost release to use.
	// 2. If it is a file:// uri pointing to a binary, use the latest Mattermost release and replace
	//    its binary with the binary pointed to by the file:// uri.
	// 3. If it is a file:// pointing to a tar.gz, use that as the Mattermost release.
	MattermostDownloadURL string `default:"https://latest.mattermost.com/mattermost-enterprise-linux" validate:"url"`
	// Path to the Mattermost EE license file.
	MattermostLicenseFile string `default:"" validate:"file"`
	// Optional path to a partial Mattermost config file to be applied as patch during
	// app server deployment.
	MattermostConfigPatchFile string `default:""`
	// Mattermost instance sysadmin e-mail.
	AdminEmail string `default:"sysadmin@sample.mattermost.com" validate:"email"`
	// Mattermost instance sysadmin user name.
	AdminUsername string `default:"sysadmin" validate:"notempty"`
	// Mattermost instance sysadmin password.
	AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
	// URL from where to download load-test-ng binaries and configuration files.
	// The configuration files provided in the package will be overridden in
	// the deployment process.
	LoadTestDownloadURL   string `default:"https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.21.0/mattermost-load-test-ng-v1.21.0-linux-amd64.tar.gz" validate:"url"`
	ElasticSearchSettings ElasticSearchSettings
	RedisSettings         RedisSettings
	JobServerSettings     JobServerSettings
	LogSettings           logger.Settings
	Report                report.Config
	// Directory under which the .terraform directory and state files are managed.
	// It will be created if it does not exist
	TerraformStateDir string `default:"/var/lib/mattermost-load-test-ng" validate:"notempty"`
	// URI of an S3 bucket whose contents are copied to the bucket created in the deployment
	S3BucketDumpURI string `default:"" validate:"s3uri"`
	// An optional URI to a MM server database dump file
	// to be loaded before running the load-test.
	// The file is expected to be gzip compressed.
	// This can also point to a local file if prefixed with "file://".
	// In such case, the dump file will be uploaded to the app servers.
	DBDumpURI string `default:""`
	// DBExtraSQL are optional URIs to SQL files containing SQL statements to be applied
	// to the Mattermost database.
	// The file is expected to be gzip compressed.
	// This can also point to a local file if prefixed with "file://".
	DBExtraSQL []string `default:"[]"`
	// An optional host name that will:
	//   - Override the SiteUrl
	//   - Point to the proxy IP via a new entry in the server's /etc/hosts file
	SiteURL string `default:"ltserver"`
	// ServerURL is the URL of the Mattermost server URL that the agent client will use to connect to the
	// Mattermost servers. This is used to override the server URL in the agent's config in case there's a
	// proxy in front of the Mattermost server.
	ServerURL string `default:""`
	// UsersFilePath specifies the path to an optional file containing a list of credentials for the controllers
	// to use. If present, it is used to automatically upload it to the agents and override the agent's config's
	// own UsersFilePath.
	UsersFilePath string `default:""`
	// PyroscopeSettings contains the settings for configuring the continuous profiling through Pyroscope
	PyroscopeSettings PyroscopeSettings
	// StorageSizes specifies the sizes of the disks for each instance type
	StorageSizes StorageSizes
}

type StorageSizes struct {
	// Size, in GiB, for the storage of the agents instances
	Agent int `default:"10"`
	// Size, in GiB, for the storage of the proxy instance
	Proxy int `default:"10"`
	// Size, in GiB, for the storage of the app instances
	App int `default:"10"`
	// Size, in GiB, for the storage of the metrics instance
	Metrics int `default:"50"`
	// Size, in GiB, for the storage of the job server instances
	Job int `default:"50"`
	// Size, in GiB, for the storage of the elasticsearch instances
	ElasticSearch int `default:"20"`
	// Size, in GiB, for the storage of the keycloak instances
	KeyCloak int `default:"10"`
}

// PyroscopeSettings contains flags to enable/disable the profiling
// of the different parts of the deployment.
type PyroscopeSettings struct {
	// Enable profiling of all the app instances
	EnableAppProfiling bool `default:"true"`
	// Enable profiling of all the agent instances
	EnableAgentProfiling bool `default:"true"`
	// Set the pprof block profile rate.
	// This value applies to both agent and Mattermost server processes.
	BlockProfileRate int `default:"0"`
}

// TerraformDBSettings contains the necessary data
// to configure an instance to be deployed
// and provisioned.
type TerraformDBSettings struct {
	// Number of DB instances.
	InstanceCount int `default:"1" validate:"range:[0,)"`
	// Type of the DB instance.
	InstanceType string `default:"db.r7g.large" validate:"notempty"`
	// Type of the DB instance - postgres or mysql.
	InstanceEngine string `default:"aurora-postgresql" validate:"oneof:{aurora-mysql, aurora-postgresql}"`
	// Username to connect to the DB.
	UserName string `default:"mmuser" validate:"notempty"`
	// Password to connect to the DB.
	Password string `default:"mostest80098bigpass_" validate:"notempty"`
	// If set to true enables performance insights for the created DB instances.
	EnablePerformanceInsights bool `default:"true"`
	// A list of DB specific parameters to use for the created instance.
	DBParameters DBParameters
	// ClusterIdentifier indicates to point to an existing cluster
	ClusterIdentifier string `default:""`
	// DBName specifies the name of the database.
	// If ClusterIdentifier is not empty, DBName should be set to the name of the database in such cluster.
	// If ClusterIdentifier is empty, the database created will use DBName as its name.
	DBName string `default:""`
}

// ExternalDBSettings contains the necessary data
// to configure an instance to be deployed
// and provisioned.
type ExternalDBSettings struct {
	// Mattermost database driver
	DriverName string `default:"" validate:"oneof:{mysql, postgres, cockroach}"`
	// DSN to connect to the database
	DataSource string `default:""`
	// DSN to connect to the database replicas
	DataSourceReplicas []string `default:""`
	// DSN to connect to the database search replicas
	DataSourceSearchReplicas []string `default:""`
}

// ExternalBucketSettings contains the necessary data
// to connect to an existing S3 bucket.
type ExternalBucketSettings struct {
	AmazonS3AccessKeyId     string `default:""`
	AmazonS3SecretAccessKey string `default:""`
	AmazonS3Bucket          string `default:""`
	AmazonS3PathPrefix      string `default:""`
	AmazonS3Region          string `default:"us-east-1"`
	AmazonS3Endpoint        string `default:"s3.amazonaws.com"`
	AmazonS3SSL             bool   `default:"true"`
	AmazonS3SignV2          bool   `default:"false"`
	AmazonS3SSE             bool   `default:"false"`
}

// ExternalAuthProviderSettings contains the necessary data
// to configure an external auth provider.
type ExternalAuthProviderSettings struct {
	// Enabled is set to true if the external auth provider should be enabled.
	Enabled bool `default:"false"`
	// DevelopmentMode is set to true if the keycloak instance should be started in development mode.
	DevelopmentMode bool `default:"true"`
	// KeycloakVersion is the version of keycloak to deploy.
	KeycloakVersion string `default:"24.0.2"`
	// KeycloakInstanceType is the type of the EC2 instance for keycloak.
	InstanceType string `default:"c7i.xlarge"`
	// KeycloakAdminUser is the username of the keycloak admin interface (admin on the master realm)
	KeycloakAdminUser string `default:"mmuser" validate:"notempty"`
	// KeycloakAdminPassword is the password of the keycloak admin interface (admin on the master realm)
	KeycloakAdminPassword string `default:"mmpass" validate:"notempty"`
	// KeycloakRealmFilePath is the path to the realm file to be uploaded to the keycloak instance.
	// If empty, a default realm file will be used.
	KeycloakRealmFilePath string `default:""`
	// KeycloakDBDumpURI
	// An optional URI to a keycloak database dump file to be uploaded on environment
	// creation.
	// The file is expected to be gzip compressed.
	// This can also point to a local file if prefixed with "file://".
	KeycloakDBDumpURI string `default:""`
	// GenerateUsersCount is the number of users to generate in the keycloak instance.
	GenerateUsersCount int `default:"0" validate:"range:[0,)"`
	// KeycloakRealmName is the name of the realm to be used in Mattermost. Must exist in the keycloak instance.
	// It is used when creating users and to properly set the OpenID configuration in Mattermost.
	KeycloakRealmName string `default:"mattermost"`
	// KeycloakClientID is the client id to be used in Mattermost from the above realm.
	// Must exist in the keycloak instance
	KeycloakClientID string `default:"mattermost-openid"`
	// KeycloakClientSecret is the client secret from the above realm to be used in Mattermost.
	// Must exist in the keycloak instance
	KeycloakClientSecret string `default:"qbdUj4dacwfa5sIARIiXZxbsBFoopTyf"`
	// KeycloakSAMLClientID is the client id to be used in Mattermost from the SAML client.
	KeycloakSAMLClientID string `default:"mattermost-saml"`
	// KeycloakSAMLClientSecret is the SAML client secret from the above realm to be used in Mattermost.
	KeycloakSAMLClientSecret string `default:"9c2edd74-9e20-454d-8cc2-0714e43f5f7e"`
}

// ElasticSearchSettings contains the necessary data
// to configure an ElasticSearch instance to be deployed
// and provisioned.
type ElasticSearchSettings struct {
	// Elasticsearch instances number.
	InstanceCount int
	// Elasticsearch instance type to be created.
	InstanceType string
	// Elasticsearch version to be deployed.
	Version string `default:"Elasticsearch_7.10"`
	// Id of the VPC associated with the instance to be created.
	VpcID string
	// Set to true if the AWSServiceRoleForAmazonElasticsearchService role should be created.
	CreateRole bool
	// SnapshotRepository is the name of the S3 bucket where the snapshot to restore lives.
	SnapshotRepository string
	// SnapshotName is the name of the snapshot to restore.
	SnapshotName string
	// RestoreTimeoutMinutes is the maximum time, in minutes, that the system will wait for the snapshot to be restored.
	RestoreTimeoutMinutes int `default:"45" validate:"range:[0,)"`
	// ClusterTimeoutMinutes is the maximum time, in minutes, that the system will wait for the cluster status to get green.
	ClusterTimeoutMinutes int `default:"45" validate:"range:[0,)"`
}

type RedisSettings struct {
	// Enabled indicates whether to add Redis or not.
	Enabled bool
	// NodeType indicates the instance type.
	NodeType string `default:"cache.m7g.2xlarge"`
	// ParameterGroupName indicates the parameter group to attach.
	ParameterGroupName string `default:"default.redis7"`
	// EngineVersion indicates the engine version.
	EngineVersion string `default:"7.1"`
}

// JobServerSettings contains the necessary data to deploy a job
// server.
type JobServerSettings struct {
	// Job server instances count.
	InstanceCount int `default:"0" validate:"range:[0,1]"`
	// Job server instance type to be created.
	InstanceType string `default:"c7i.xlarge"`
}

// DBParameter contains info regarding a single RDS DB specific parameter.
type DBParameter struct {
	// The unique name for the parameter.
	Name string `validate:"notempty"`
	// The value for the parameter.
	Value string `validate:"notempty"`
	// The apply method for the parameter. Can be either "immediate" or
	// "pending-reboot". It depends on the db engine used and parameter type.
	ApplyMethod string `validate:"oneof:{immediate, pending-reboot}"`
}

type DBParameters []DBParameter

func (p DBParameters) String() string {
	var b strings.Builder
	b.WriteString("[")
	for i, param := range p {
		fmt.Fprintf(&b, `{name = %q, value = %q, apply_method = %q}`, param.Name, param.Value, param.ApplyMethod)
		if i != len(p)-1 {
			b.WriteString(",")
		}
	}
	b.WriteString("]")
	return b.String()
}

// IsValid reports whether a given deployment config is valid or not.
func (c *Config) IsValid() error {
	if !checkPrefix(c.MattermostDownloadURL) {
		return fmt.Errorf("mattermost download url is not in correct format: %q", c.MattermostDownloadURL)
	}

	if !checkPrefix(c.LoadTestDownloadURL) {
		return fmt.Errorf("load-test download url is not in correct format: %q", c.LoadTestDownloadURL)
	}

	if err := c.validateElasticSearchConfig(); err != nil {
		return err
	}

	if err := c.validateProxyConfig(); err != nil {
		return err
	}

	if err := c.validateDBName(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateProxyConfig() error {
	if c.AppInstanceCount > 1 && c.ProxyInstanceCount < 1 && c.ServerURL == "" {
		return fmt.Errorf("the deployment will create more than one app node, but no proxy is being deployed and no external proxy has been configured: either set ProxyInstanceCount to 1, or set ServerURL to the URL of an external proxy")
	}
	return nil
}

// DBName returns the database name for the deployment.
func (c *Config) DBName() string {
	if c.TerraformDBSettings.DBName != "" {
		return c.TerraformDBSettings.DBName
	}
	return c.ClusterName + "db"
}

func checkPrefix(str string) bool {
	return strings.HasPrefix(str, "https://") ||
		strings.HasPrefix(str, "http://") ||
		strings.HasPrefix(str, "file://")
}

func (c *Config) validateElasticSearchConfig() error {
	if c.ElasticSearchSettings.InstanceCount == 0 {
		return nil
	}

	if (c.ElasticSearchSettings != ElasticSearchSettings{}) {
		if c.ElasticSearchSettings.VpcID == "" {
			return errors.New("VpcID must be set in order to create an Elasticsearch instance")
		}

		domainName := c.ClusterName + "-es"
		if !esDomainNameRe.Match([]byte(domainName)) {
			return fmt.Errorf("Elasticsearch domain name must start with a lowercase alphabet and be at least " +
				"3 and no more than 28 characters long. Valid characters are a-z (lowercase letters), 0-9, and - " +
				"(hyphen). Current value is \"" + domainName + "\"")
		}

	}

	if !strings.HasPrefix(c.ElasticSearchSettings.Version, "OpenSearch") {
		return fmt.Errorf("Incorrect engine version: %s. Must start with %q", c.ElasticSearchSettings.Version, "OpenSearch")
	}

	if c.ElasticSearchSettings.SnapshotRepository == "" {
		return fmt.Errorf("Empty SnapshotRepository. Must supply a value")
	}

	if c.ElasticSearchSettings.SnapshotName == "" {
		return fmt.Errorf("Empty SnapshotName. Must supply a value")
	}

	return nil
}

func (c *Config) validateDBName() error {
	if c.TerraformDBSettings.ClusterIdentifier == "" {
		return nil
	}

	if c.TerraformDBSettings.DBName == "" {
		return fmt.Errorf("TerraformDBSettings.ClusterIdentifier is specified but TerraformDBSettings.DBName is empty: TerraformDBSettings.DBName should be set to the name of the database contained in the cluster specified by TerraformDBSettings.ClusterIdentifier")
	}

	return nil
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFrom(configFilePath, "./config/deployer.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
