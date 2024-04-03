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
	AppInstanceType string `default:"c5.xlarge" validate:"notempty"`
	// Number of agents, first agent and coordinator will share the same instance.
	AgentInstanceCount int `default:"2" validate:"range:[0,)"`
	// Type of the EC2 instance for agent.
	AgentInstanceType string `default:"c5.xlarge" validate:"notempty"`
	// Logs the command output (stdout & stderr) to home directory.
	EnableAgentFullLogs bool `default:"true"`
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
	// URL from where to download Mattermost release.
	// This can also point to a local binary path if the user wants to run loadtest
	// on a custom build. The path should be prefixed with "file://". In that case,
	// only the binary gets replaced, and the rest of the build comes from the latest
	// stable release.
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
	LoadTestDownloadURL   string `default:"https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.15.0-rc2/mattermost-load-test-ng-v1.15.0-rc2-linux-amd64.tar.gz" validate:"url"`
	ElasticSearchSettings ElasticSearchSettings
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
	// An optional host name that will:
	//   - Override the SiteUrl
	//   - Point to the proxy IP via a new entry in the server's /etc/hosts file
	SiteURL string `default:"ltserver"`
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
}

// TerraformDBSettings contains the necessary data
// to configure an instance to be deployed
// and provisioned.
type TerraformDBSettings struct {
	// Number of DB instances.
	InstanceCount int `default:"1" validate:"range:[0,)"`
	// Type of the DB instance.
	InstanceType string `default:"db.r6g.large" validate:"notempty"`
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
	InstanceCount          int          `default:"0" validate:"range:[0,1]"`
	DevelopmentMode        bool         `default:"true"`
	KeycloakVersion        string       `default:"24.0.2"`
	InstanceType           string       `default:"c5.xlarge"`
	KeycloakAdminUser      string       `default:"mmuser" validate:"notempty"`
	KeycloakAdminPassword  string       `default:"mmpass" validate:"notempty"`
	KeycloakRealmFilePath  string       `default:""`
	GenerateUsersCount     int          `default:"0" validate:"range:[0,)"`
	KeycloakRealmName      string       `default:"mattermost"`
	KeycloakClientID       string       `default:"mattermost-openid"`
	KeycloakClientSecret   string       `default:"qbdUj4dacwfa5sIARIiXZxbsBFoopTyf"`
	DatabaseInstanceCount  int          `default:"1" validate:"range:[0,1]"`
	DatabaseInstanceEngine string       `default:"aurora-postgresql"`
	DatabaseInstanceType   string       `default:"db.r6g.large"`
	DatabaseUsername       string       `default:"mmuser"`
	DatabasePassword       string       `default:"mmpassword"`
	DatabaseParameters     DBParameters `default:"[]"`
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
	Version string `default:"Elasticsearch_7.10" validate:"prefix:Elasticsearch_"`
	// Id of the VPC associated with the instance to be created.
	VpcID string
	// Set to true if the AWSServiceRoleForAmazonElasticsearchService role should be created.
	CreateRole bool
}

// JobServerSettings contains the necessary data to deploy a job
// server.
type JobServerSettings struct {
	// Job server instances count.
	InstanceCount int `default:"0" validate:"range:[0,1]"`
	// Job server instance type to be created.
	InstanceType string `default:"c5.xlarge"`
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

	if err := c.validateDBName(); err != nil {
		return err
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
	if (c.ElasticSearchSettings != ElasticSearchSettings{}) {
		if c.ElasticSearchSettings.InstanceCount > 1 {
			return errors.New("it is not possible to create more than 1 instance of Elasticsearch")
		}

		if c.ElasticSearchSettings.InstanceCount > 0 && c.ElasticSearchSettings.VpcID == "" {
			return errors.New("VpcID must be set in order to create an Elasticsearch instance")
		}

		domainName := c.ClusterName + "-es"
		if !esDomainNameRe.Match([]byte(domainName)) {
			return fmt.Errorf("Elasticsearch domain name must start with a lowercase alphabet and be at least " +
				"3 and no more than 28 characters long. Valid characters are a-z (lowercase letters), 0-9, and - " +
				"(hyphen). Current value is \"" + domainName + "\"")
		}

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

	if err := defaults.ReadFromJSON(configFilePath, "./config/deployer.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
