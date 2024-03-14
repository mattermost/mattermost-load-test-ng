package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigIsValid(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			MattermostDownloadURL: "https://latest.mattermost.com/mattermost-enterprise-linux",
			LoadTestDownloadURL:   "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.15.0-rc2/mattermost-load-test-ng-v1.15.0-rc2-linux-amd64.tar.gz",
		}
	}

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
