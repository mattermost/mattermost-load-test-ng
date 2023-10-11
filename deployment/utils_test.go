// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildLoadDBDumpCmd(t *testing.T) {
	tcs := []struct {
		name     string
		dumpName string
		dbInfo   DBSettings
		cmd      string
		err      string
	}{
		{
			name:     "invalid engine",
			dumpName: "invalid.dump.sql",
			dbInfo: DBSettings{
				Engine:   "invalid",
				UserName: "mmuser",
				Password: "mostest",
				DBName:   "mattermost",
				Host:     "localhost",
			},
			err: "invalid db engine invalid",
		},
		{
			name:     "mysql",
			dumpName: "mysql.dump.sql.xz",
			dbInfo: DBSettings{
				Engine:   "aurora-mysql",
				UserName: "mmuser",
				Password: "mostest",
				DBName:   "mattermost",
				Host:     "localhost",
			},
			cmd: `zcat mysql.dump.sql.xz | mysql -h localhost -u mmuser -pmostest mattermost && mysql -h localhost -u mmuser -pmostest mattermost -e "DELETE FROM Systems WHERE Name = 'ActiveLicenseId'; DELETE FROM Licenses;"`,
		},
		{
			name:     "psql",
			dumpName: "psql.dump.sql.xz",
			dbInfo: DBSettings{
				Engine:   "aurora-postgresql",
				UserName: "mmuser",
				Password: "mostest",
				DBName:   "mattermost",
				Host:     "localhost",
			},
			cmd: `zcat psql.dump.sql.xz | psql 'postgres://mmuser:mostest@localhost/mattermost?sslmode=disable' && psql 'postgres://mmuser:mostest@localhost/mattermost?sslmode=disable' -c "DELETE FROM Systems WHERE Name = 'ActiveLicenseId'; DELETE FROM Licenses;"`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := BuildLoadDBDumpCmd(tc.dumpName, tc.dbInfo)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.cmd, cmd)
		})
	}
}
