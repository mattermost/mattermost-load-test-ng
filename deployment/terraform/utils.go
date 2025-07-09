// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/channels/utils"
)

type uploadInfo struct {
	msg     string
	srcData string
	dstPath string
}

func uploadBatch(sshc *ssh.Client, batch []uploadInfo) error {
	if sshc == nil {
		return errors.New("sshc should not be nil")
	}
	if len(batch) == 0 {
		return errors.New("batch should not be empty")
	}

	for _, info := range batch {
		if info.msg != "" {
			mlog.Info(info.msg)
		}
		rdr := strings.NewReader(info.srcData)
		if out, err := sshc.Upload(rdr, info.dstPath, true); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", info.dstPath, out, err)
		}
	}

	return nil
}

// OpenSSHFor starts a ssh connection to the resource
func (t *Terraform) OpenSSHFor(resource string) error {
	cmd, err := t.makeCmdForResource(resource)
	if err != nil {
		return fmt.Errorf("failed to make cmd for resource: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

// RunSSHCommand runs a command on a given machine.
func (t *Terraform) RunSSHCommand(resource string, args []string) error {
	cmd, err := t.makeCmdForResource(resource)
	if err != nil {
		return fmt.Errorf("failed to make cmd for resource: %w", err)
	}
	cmd.Args = append(cmd.Args, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func (t *Terraform) makeCmdForResource(resource string) (*exec.Cmd, error) {
	output, err := t.Output()
	if err != nil {
		return nil, fmt.Errorf("could not parse output: %w", err)
	}

	// Match against the agent names, or the reserved "coordinator" keyword referring to the
	// first agent.
	for i, agent := range output.Agents {
		if resource == agent.Tags.Name || (i == 0 && resource == "coordinator") {
			return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, agent.GetConnectionIP())), nil
		}
	}

	// Match against the instance names.
	for _, instance := range output.Instances {
		if resource == instance.Tags.Name {
			return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, instance.GetConnectionIP())), nil
		}
	}

	// Match against the job server names.
	for _, instance := range output.JobServers {
		if resource == instance.Tags.Name {
			return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, instance.GetConnectionIP())), nil
		}
	}

	// Match against proxy names
	for _, inst := range output.Proxies {
		if resource == inst.Tags.Name {
			return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, inst.GetConnectionIP())), nil
		}
	}

	// Match against the keycloak server
	if output.KeycloakServer.Tags.Name == resource {
		return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, output.KeycloakServer.GetConnectionIP())), nil
	}

	// Match against the metrics servers, as well as convenient aliases.
	switch resource {
	case "metrics", "prometheus", "grafana", output.MetricsServer.Tags.Name:
		return exec.Command("ssh", fmt.Sprintf("%s@%s", t.Config().AWSAMIUser, output.MetricsServer.GetConnectionIP())), nil
	}

	return nil, fmt.Errorf("could not find any resource with name %q", resource)
}

// OpenBrowserFor opens a web browser for the resource
func (t *Terraform) OpenBrowserFor(resource string) error {
	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}
	url := "http://"
	switch resource {
	case "grafana":
		url += output.MetricsServer.GetConnectionDNS() + ":3000"
	case "mattermost":
		if output.HasProxy() {
			url += output.Proxies[0].GetConnectionDNS()
		} else {
			url += output.Instances[0].GetConnectionDNS() + ":8065"
		}
	case "prometheus":
		url += output.MetricsServer.GetConnectionDNS() + ":9090"
	default:
		return fmt.Errorf("undefined resource :%q", resource)
	}
	fmt.Printf("Opening %s...\n", url)
	return openBrowser(url)
}

func openBrowser(url string) (err error) {
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = errors.New("unsupported platform")
	}
	return
}

func validateLicense(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	validator := &utils.LicenseValidatorImpl{}
	licenseStr, err := validator.ValidateLicense(data)
	// If we cannot validate the license, we can test using another service
	// environment to inform the user whether that's a possible solution
	if err != nil {
		currentValue := os.Getenv("MM_SERVICEENVIRONMENT")
		defer func() { os.Setenv("MM_SERVICEENVIRONMENT", currentValue) }()

		// Pick a different environment
		newValue := model.ServiceEnvironmentTest
		if currentValue != model.ServiceEnvironmentProduction {
			newValue = model.ServiceEnvironmentProduction
		}
		os.Setenv("MM_SERVICEENVIRONMENT", newValue)

		// If the error is not nil, then the user just needs to set the
		// -service_environment flag to a different value
		if _, newEnvErr := validator.ValidateLicense(data); newEnvErr == nil {
			return fmt.Errorf("this license is valid only with a %q service environment, which is currently set to %q; try adding the -service_environment=%s flag to change it", newValue, currentValue, newValue)
		}

		// If not, we just return the (probably not very useful) error returned
		// by the validator
		return fmt.Errorf("failed to validate license: %w", err)
	}

	var license model.License
	if err := json.Unmarshal([]byte(licenseStr), &license); err != nil {
		return fmt.Errorf("failed to parse license: %w", err)
	}

	if !license.IsStarted() {
		return errors.New("license has not started")
	}

	if license.IsExpired() {
		return errors.New("license has expired")
	}

	return nil
}

func (t *Terraform) getStatePath() string {
	// Get the name of the file
	statePath := "terraform.tfstate"
	if t.id != "" {
		statePath = t.id + ".tfstate"
	}

	return statePath
}

func fillConfigTemplate(configTmpl string, data map[string]any) (string, error) {
	var buf bytes.Buffer
	tmpl := template.New("template")
	tmpl, err := tmpl.Parse(configTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

func (t *Terraform) getParams() []string {
	return []string{
		"-var", fmt.Sprintf("aws_profile=%s", t.config.AWSProfile),
		"-var", fmt.Sprintf("aws_region=%s", t.config.AWSRegion),
		"-var", fmt.Sprintf("aws_az=%s", t.config.AWSAvailabilityZone),
		"-var", fmt.Sprintf("aws_ami=%s", t.config.AWSAMI),
		"-var", fmt.Sprintf("aws_ami_user=%s", t.config.AWSAMIUser),
		"-var", fmt.Sprintf("operating_system_kind=%s", t.config.OperatingSystemKind),
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("cluster_vpc_id=%s", t.config.ClusterVpcID),
		"-var", fmt.Sprintf(`cluster_subnet_ids=%s`, t.config.ClusterSubnetIDs),
		"-var", fmt.Sprintf("connection_type=%s", t.config.ConnectionType),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("app_instance_type=%s", t.config.AppInstanceType),
		"-var", fmt.Sprintf("app_attach_iam_profile=%s", t.config.AppAttachIAMProfile),
		"-var", fmt.Sprintf("agent_instance_count=%d", t.config.AgentInstanceCount),
		"-var", fmt.Sprintf("agent_instance_type=%s", t.config.AgentInstanceType),
		"-var", fmt.Sprintf("agent_allocate_public_ip_address=%t", t.config.AgentAllocatePublicIPAddress),
		"-var", fmt.Sprintf("es_instance_count=%d", t.config.ElasticSearchSettings.InstanceCount),
		"-var", fmt.Sprintf("es_instance_type=%s", t.config.ElasticSearchSettings.InstanceType),
		"-var", fmt.Sprintf("es_version=%s", t.config.ElasticSearchSettings.Version),
		"-var", fmt.Sprintf("es_create_role=%t", t.config.ElasticSearchSettings.CreateRole),
		"-var", fmt.Sprintf("es_snapshot_repository=%s", t.config.ElasticSearchSettings.SnapshotRepository),
		"-var", fmt.Sprintf("es_zone_awareness_enabled=%t", t.config.ElasticSearchSettings.ZoneAwarenessEnabled),
		"-var", fmt.Sprintf("es_zone_awarness_availability_zone_count=%d", t.config.ElasticSearchSettings.ZoneAwarenessAZCount),
		"-var", fmt.Sprintf("es_enable_cloudwatch_logs=%t", t.config.ElasticSearchSettings.EnableCloudwatchLogs),
		"-var", fmt.Sprintf("proxy_instance_count=%d", t.config.ProxyInstanceCount),
		"-var", fmt.Sprintf("proxy_instance_type=%s", t.config.ProxyInstanceType),
		"-var", fmt.Sprintf("proxy_allocate_public_ip_address=%t", t.config.ProxyAllocatePublicIPAddress),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.TerraformDBSettings.InstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.TerraformDBSettings.InstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.TerraformDBSettings.InstanceType),
		"-var", fmt.Sprintf("db_cluster_identifier=%s", t.config.TerraformDBSettings.ClusterIdentifier),
		"-var", fmt.Sprintf("db_username=%s", t.config.TerraformDBSettings.UserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.TerraformDBSettings.Password),
		"-var", fmt.Sprintf("db_enable_performance_insights=%t", t.config.TerraformDBSettings.EnablePerformanceInsights),
		"-var", fmt.Sprintf("db_parameters=%s", t.config.TerraformDBSettings.DBParameters),
		"-var", fmt.Sprintf("keycloak_enabled=%v", t.config.ExternalAuthProviderSettings.Enabled),
		"-var", fmt.Sprintf("keycloak_development_mode=%v", t.config.ExternalAuthProviderSettings.DevelopmentMode),
		"-var", fmt.Sprintf("keycloak_instance_type=%s", t.config.ExternalAuthProviderSettings.InstanceType),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("job_server_instance_count=%d", t.config.JobServerSettings.InstanceCount),
		"-var", fmt.Sprintf("job_server_instance_type=%s", t.config.JobServerSettings.InstanceType),
		"-var", fmt.Sprintf("s3_bucket_dump_uri=%s", t.config.S3BucketDumpURI),
		"-var", fmt.Sprintf("s3_external_bucket_name=%s", t.config.ExternalBucketSettings.AmazonS3Bucket),
		"-var", fmt.Sprintf("block_device_sizes_agent=%d", t.config.StorageSizes.Agent),
		"-var", fmt.Sprintf("block_device_sizes_proxy=%d", t.config.StorageSizes.Proxy),
		"-var", fmt.Sprintf("block_device_sizes_app=%d", t.config.StorageSizes.App),
		"-var", fmt.Sprintf("block_device_sizes_metrics=%d", t.config.StorageSizes.Metrics),
		"-var", fmt.Sprintf("block_device_sizes_job=%d", t.config.StorageSizes.Job),
		"-var", fmt.Sprintf("block_device_sizes_elasticsearch=%d", t.config.StorageSizes.ElasticSearch),
		"-var", fmt.Sprintf("block_device_sizes_keycloak=%d", t.config.StorageSizes.KeyCloak),
		"-var", fmt.Sprintf("redis_enabled=%t", t.config.RedisSettings.Enabled),
		"-var", fmt.Sprintf("redis_node_type=%s", t.config.RedisSettings.NodeType),
		"-var", fmt.Sprintf("redis_param_group_name=%s", t.config.RedisSettings.ParameterGroupName),
		"-var", fmt.Sprintf("redis_engine_version=%s", t.config.RedisSettings.EngineVersion),
		"-var", fmt.Sprintf("custom_tags=%s", t.config.CustomTags),
		"-var", fmt.Sprintf("enable_metrics_instance=%t", t.config.EnableMetricsInstance),
		"-var", fmt.Sprintf("metrics_instance_type=%s", t.config.MetricsInstanceType),
	}
}

func (t *Terraform) getClusterDSN() (string, error) {
	switch t.config.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		return "postgres://" + t.config.TerraformDBSettings.UserName + ":" + t.config.TerraformDBSettings.Password + "@" + t.output.DBWriter() + "/" + t.config.DBName() + "?sslmode=disable", nil

	case "aurora-mysql":
		return t.config.TerraformDBSettings.UserName + ":" + t.config.TerraformDBSettings.Password + "@tcp(" + t.output.DBWriter() + ")/" + t.config.DBName() + "?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s", nil
	default:
		return "", errors.New("unsupported database engine")
	}
}

func (t *Terraform) getAsset(filename string) string {
	return filepath.Join(t.config.TerraformStateDir, filename)
}

// getServerURL returns the URL of the server to be used for testing.
// server URL priority:
// 1. ServerURL
// 2. SiteURL
// 3. Proxy IP
// 4. First app server IP
func getServerURL(output *Output, deploymentConfig *deployment.Config) string {
	if deploymentConfig.ServerURL != "" {
		return deploymentConfig.ServerScheme + "://" + deploymentConfig.ServerURL
	}

	url := output.Instances[0].GetConnectionIP()
	if deploymentConfig.SiteURL != "" {
		url = deploymentConfig.SiteURL
	}

	if !output.HasProxy() {
		url = url + ":8065"
	} else if deploymentConfig.SiteURL == "" {
		// It's an error to have siteURL empty and set multiple proxies. (see (c *Config) validateProxyConfig)
		// So we can safely take the IP of the first entry.
		url = output.Proxies[0].PrivateIP
	}

	return deploymentConfig.ServerScheme + "://" + url
}

// Durations for the expiry of the AWS Role credentials and for its refresh
// interval, which, out of an abundance of caution, is a bit shorter than half
// the expiry duration of the role.
const (
	// One hour is the maximum allowed by role chaining
	AWSRoleTokenDuration        = 1 * time.Hour
	AWSRoleTokenRefreshInterval = 25 * time.Minute
)

// InitCreds authenticates into AWS using the default credentials chain,
// optionally assuming the AWS role if its ARN is provided in the configuration.
// In that case, it also starts a goroutine that will refresh the credentials to
// maintain them updated.
func (t *Terraform) InitCreds() error {
	regionOpt := awsconfig.WithRegion(t.config.AWSRegion)
	var opts []func(*awsconfig.LoadOptions) error

	opts = append(opts, regionOpt)

	if t.config.AWSProfile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(t.config.AWSProfile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return fmt.Errorf("unable to load default config: %w", err)
	}

	if t.config.AWSRoleARN != "" {
		// See https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds#hdr-Assume_Role
		// Create the credentials from AssumeRoleProvider to assume the role
		// identified by AWSRoleARN, and use them in the loaded config
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, t.config.AWSRoleARN)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	t.awsCfg = cfg

	if t.config.AWSRoleARN != "" {
		// Refresh the credentials once so that the time of the interval below
		// starts now
		if err := t.refreshAWSCredentialsFromRole(); err != nil {
			return fmt.Errorf("unable to refresh token: %w", err)
		}

		// Start a goroutine refreshing the credentials every AWSRoleTokenRefreshInterval.
		// If it fails, exit the goroutine, since it needs the previous credentials to be
		// valid for the refresh logic to work.
		go func() {
			ticker := time.Tick(AWSRoleTokenRefreshInterval)
			for range ticker {
				if err := t.refreshAWSCredentialsFromRole(); err != nil {
					mlog.Error("unable to refresh token, stopping goroutine", mlog.Err(err))
					return
				}
			}
		}()
	}

	return nil
}

// refreshAWSCredentialsFromRole refreshes the current credentials by assuming
// the role again, with an expiry duration of one hour, which is the maximum
// allowed when using role chaining
func (t *Terraform) refreshAWSCredentialsFromRole() error {
	t.awsCfgMut.Lock()
	defer t.awsCfgMut.Unlock()

	if t.config.AWSRoleARN == "" {
		return fmt.Errorf("AWSRoleARN must be non-empty")
	}

	// Test current credentials before refresh: if they are not valid, we
	// cannot refresh
	if _, err := t.awsCfg.Credentials.Retrieve(context.Background()); err != nil {
		return fmt.Errorf("failed to retrieve current credentials: %w", err)
	}

	// Create new STS service and assume role again
	stsSvc := sts.NewFromConfig(t.awsCfg)
	creds := stscreds.NewAssumeRoleProvider(stsSvc, t.config.AWSRoleARN, func(opt *stscreds.AssumeRoleOptions) { opt.Duration = AWSRoleTokenDuration })
	newCredCache := aws.NewCredentialsCache(creds)

	// Test new credentials
	newCreds, err := newCredCache.Retrieve(context.Background())
	if err != nil {
		return fmt.Errorf("failed to assume role %q with new credentials: %w", t.config.AWSRoleARN, err)
	}

	// Update the stored configuration with the new credentials
	mlog.Info("successfully assumed role", mlog.Time("expiry", newCreds.Expires))
	t.awsCfg.Credentials = newCredCache

	// Further verify the credentials work with a test STS call
	if _, err := stsSvc.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{}); err != nil {
		return fmt.Errorf("new credentials failed STS test: %w", err)
	}

	return nil
}

// GetAWSConfig returns the AWS config, using the profile configured in the
// deployer if present, and defaulting to the default credential chain otherwise.
// If a role ARN is provided, it will assume that role.
func (t *Terraform) GetAWSConfig() (aws.Config, error) {
	t.awsCfgMut.Lock()
	defer t.awsCfgMut.Unlock()
	return t.awsCfg.Copy(), nil
}

// GetAWSCreds returns the AWS config, using the profile configured in the
// deployer if present, and defaulting to the default credential chain otherwise
func (t *Terraform) GetAWSCreds() (aws.Credentials, error) {
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return aws.Credentials{}, err
	}

	return cfg.Credentials.Retrieve(context.Background())
}

// ExpandWithUser replaces {{.Username}} in a path template with the provided username
func (t *Terraform) ExpandWithUser(path string) string {
	data := map[string]any{
		"Username": t.config.AWSAMIUser,
	}
	result, err := fillConfigTemplate(path, data)
	if err != nil {
		// If there's an error with the template, return the original path
		return path
	}
	return result
}

// generatePseudoRandomPassword returns a pseudo-random string containing
// lower-case letters, upper-case letter and numbers
func generatePseudoRandomPassword(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789")
	s := make([]rune, length)
	for j := 0; j < length; j++ {
		s[j] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}
