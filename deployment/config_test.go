package deployment

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/stretchr/testify/require"
)

func TestConfigIsValid(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			MattermostDownloadURL: "https://latest.mattermost.com/mattermost-enterprise-linux",
			LoadTestDownloadURL:   "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.22.0/mattermost-load-test-ng-v1.22.0-linux-amd64.tar.gz",
		}
	}

	t.Run("paths", func(t *testing.T) {
		t.Run("MattermostDownloadUrl can be an url", func(t *testing.T) {
			c := baseConfig()

			require.NoError(t, c.IsValid())
		})

		t.Run("MattermostDownloadUrl can be a path", func(t *testing.T) {
			c := baseConfig()
			c.MattermostDownloadURL = "file:///some/path"

			require.NoError(t, c.IsValid())
		})

		t.Run("MattermostDownloadUrl must be an url or a file", func(t *testing.T) {
			c := baseConfig()
			c.MattermostDownloadURL = "/some/path"

			require.Error(t, c.IsValid())
		})
	})

	t.Run("DBName is valid", func(t *testing.T) {
		t.Run("empty ClusterIdentifier and empty DBName is valid", func(t *testing.T) {
			c := baseConfig()
			c.TerraformDBSettings.ClusterIdentifier = ""
			c.TerraformDBSettings.DBName = ""

			require.NoError(t, c.IsValid())
		})

		t.Run("empty ClusterIdentifier and non-empty DBName is valid", func(t *testing.T) {
			c := baseConfig()
			c.TerraformDBSettings.ClusterIdentifier = ""
			c.TerraformDBSettings.DBName = "db"

			require.NoError(t, c.IsValid())
		})

		t.Run("non-empty ClusterIdentifier and empty DBName is not valid", func(t *testing.T) {
			c := baseConfig()
			c.TerraformDBSettings.ClusterIdentifier = "cluster"
			c.TerraformDBSettings.DBName = ""

			require.Error(t, c.IsValid())
		})

		t.Run("non-empty ClusterIdentifier and non-empty DBName is valid", func(t *testing.T) {
			c := baseConfig()
			c.TerraformDBSettings.ClusterIdentifier = "cluster"
			c.TerraformDBSettings.DBName = "db"

			require.NoError(t, c.IsValid())
		})
	})
}

func TestValidateElasticSearchConfig(t *testing.T) {
	baseValidConfig := func() Config {
		return Config{
			ClusterVpcID:          "vpc-01234567890abcdef",
			ClusterName:           "clustername",
			MattermostDownloadURL: "https://latest.mattermost.com/mattermost-enterprise-linux",
			LoadTestDownloadURL:   "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.22.0/mattermost-load-test-ng-v1.22.0-linux-amd64.tar.gz",
			ElasticSearchSettings: ElasticSearchSettings{
				InstanceCount:      1,
				Version:            "OpenSearch_2.7",
				SnapshotRepository: "somerepo",
				SnapshotName:       "somename",
			},
		}
	}

	t.Run("valid config", func(t *testing.T) {
		cfg := baseValidConfig()
		require.NoError(t, cfg.validateElasticSearchConfig())
	})

	t.Run("valid instance count", func(t *testing.T) {
		cfg := baseValidConfig()

		cfg.ElasticSearchSettings.InstanceCount = 1
		require.NoError(t, cfg.validateElasticSearchConfig())

		cfg.ElasticSearchSettings.InstanceCount = 42
		require.NoError(t, cfg.validateElasticSearchConfig())
	})

	t.Run("invalid VPC ID", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.ClusterVpcID = ""
		require.Error(t, cfg.validateElasticSearchConfig())
	})

	t.Run("invalid domain name for ES", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.ClusterName = "InvalidClusterNameForES!@#$"

		require.Error(t, cfg.validateElasticSearchConfig())
	})

	t.Run("invalid domain name for ES but validation passes because InstanceCount == 0", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.ClusterName = "InvalidClusterNameForES!@#$"
		cfg.ElasticSearchSettings.InstanceCount = 0

		require.NoError(t, cfg.validateElasticSearchConfig())
	})
}

func TestTerraformMapString(t *testing.T) {
	var nilMap TerraformMap
	emptyMap := make(TerraformMap)

	testCases := []struct {
		actual    TerraformMap
		expected  string
		expected2 string
	}{
		{
			actual: TerraformMap{
				"uno": "1",
			},
			expected: "{uno = \"1\"}",
		},
		{
			actual: TerraformMap{
				"uno": "1",
				"dos": "2",
			},
			expected:  "{uno = \"1\", dos = \"2\"}",
			expected2: "{dos = \"2\", uno = \"1\"}",
		},
		{
			actual:   nilMap,
			expected: "{}",
		},
		{
			actual:   emptyMap,
			expected: "{}",
		},
	}

	for _, testCase := range testCases {
		actual := testCase.actual.String()

		// map order is non deterministic
		equals := testCase.expected == actual || (testCase.expected2 != "" && testCase.expected2 == actual)
		require.True(t, equals)
	}
}

func TestClusterSubnetIDs(t *testing.T) {
	var defaultStruct ClusterSubnetIDs
	emptyStructNilSlices := ClusterSubnetIDs{}
	emptyStructEmptySlices := ClusterSubnetIDs{
		App:           []string{},
		Job:           []string{},
		Proxy:         []string{},
		Agent:         []string{},
		ElasticSearch: []string{},
		Metrics:       []string{},
		Keycloak:      []string{},
		Database:      []string{},
		Redis:         []string{},
	}

	t.Run("String()", func(t *testing.T) {
		testCases := []struct {
			actual   ClusterSubnetIDs
			expected string
		}{
			{
				actual:   defaultStruct,
				expected: `{"app":null,"job":null,"proxy":null,"agent":null,"elasticsearch":null,"metrics":null,"keycloak":null,"database":null,"redis":null}`,
			},
			{
				actual:   emptyStructNilSlices,
				expected: `{"app":null,"job":null,"proxy":null,"agent":null,"elasticsearch":null,"metrics":null,"keycloak":null,"database":null,"redis":null}`,
			},
			{
				actual:   emptyStructEmptySlices,
				expected: `{"app":[],"job":[],"proxy":[],"agent":[],"elasticsearch":[],"metrics":[],"keycloak":[],"database":[],"redis":[]}`,
			},
		}

		for _, testCase := range testCases {
			actual := testCase.actual.String()
			require.Equal(t, testCase.expected, actual)
		}
	})

	t.Run("default values", func(t *testing.T) {
		cfg := Config{}
		defaults.Set(&cfg)

		require.NotNil(t, cfg.ClusterSubnetIDs.App)
		require.Len(t, cfg.ClusterSubnetIDs.App, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Job)
		require.Len(t, cfg.ClusterSubnetIDs.Job, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Proxy)
		require.Len(t, cfg.ClusterSubnetIDs.Proxy, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Agent)
		require.Len(t, cfg.ClusterSubnetIDs.Agent, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.ElasticSearch)
		require.Len(t, cfg.ClusterSubnetIDs.ElasticSearch, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Metrics)
		require.Len(t, cfg.ClusterSubnetIDs.Metrics, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Keycloak)
		require.Len(t, cfg.ClusterSubnetIDs.Keycloak, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Database)
		require.Len(t, cfg.ClusterSubnetIDs.Database, 0)

		require.NotNil(t, cfg.ClusterSubnetIDs.Redis)
		require.Len(t, cfg.ClusterSubnetIDs.Redis, 0)

		require.NotNil(t, cfg.TerraformStateDir)
		require.Equal(t, "./ltstate", cfg.TerraformStateDir)
	})

	t.Run("String() of default values", func(t *testing.T) {
		cfg := Config{}
		defaults.Set(&cfg)

		expected := `{"app":[],"job":[],"proxy":[],"agent":[],"elasticsearch":[],"metrics":[],"keycloak":[],"database":[],"redis":[]}`
		// The bug that prompted this was that we declared String with a
		// pointer receiver, in which case fmt never calls the String method.
		// Hence the explicit test to use fmt, and the need to skip the linter
		//nolint:gosimple
		actual := fmt.Sprintf("%s", cfg.ClusterSubnetIDs)
		require.Equal(t, expected, actual)
	})

	t.Run("IsAnySet", func(t *testing.T) {
		require.False(t, defaultStruct.IsAnySet())
		require.False(t, emptyStructNilSlices.IsAnySet())
		require.False(t, emptyStructEmptySlices.IsAnySet())

		someSet := ClusterSubnetIDs{App: []string{"set"}}
		require.True(t, someSet.IsAnySet())
	})
}
