// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type Cmd struct {
	Msg     string
	Value   string
	Clients []*ssh.Client
}

type DBSettings struct {
	UserName string
	Password string
	DBName   string
	Host     string
	Engine   string
}

func dbConnString(dbInfo DBSettings) (string, error) {
	var dbConnCmd string

	switch dbInfo.Engine {
	case "aurora-postgresql":
		dbConnCmd = fmt.Sprintf("psql 'postgres://%[1]s:%[2]s@%[3]s/%[4]s?sslmode=disable'", dbInfo.UserName, dbInfo.Password, dbInfo.Host, dbInfo.DBName)
	case "aurora-mysql":
		dbConnCmd = fmt.Sprintf("mysql -h %[1]s -u %[2]s -p%[3]s %[4]s", dbInfo.Host, dbInfo.UserName, dbInfo.Password, dbInfo.DBName)
	default:
		return "", fmt.Errorf("invalid db engine %s", dbInfo.Engine)
	}

	return dbConnCmd, nil
}

// ClearLicensesCmd returns a command string to connect to the database and
// delete all rows in the Licenses table and the ActiveLicenseId row in the
// Systems table
func ClearLicensesCmd(dbInfo DBSettings) (string, error) {
	dbConnCmd, err := dbConnString(dbInfo)
	if err != nil {
		return "", err
	}

	var licenseClearCmd string
	licenseClearQuery := "DELETE FROM Systems WHERE Name = 'ActiveLicenseId'; DELETE FROM Licenses;"

	switch dbInfo.Engine {
	case "aurora-postgresql":
		licenseClearCmd = fmt.Sprintf("%s -c %q", dbConnCmd, licenseClearQuery)
	case "aurora-mysql":
		licenseClearCmd = fmt.Sprintf("%s -e %q", dbConnCmd, licenseClearQuery)
	default:
		return "", fmt.Errorf("invalid db engine %s", dbInfo.Engine)
	}

	return licenseClearCmd, nil
}

// BuildLoadDBDumpCmd returns a command string to feed the
// provided DB dump file into the database. Example:
//
//	zcat dbdump.sql | mysql/psql connection_details
func BuildLoadDBDumpCmd(dumpFilename string, dbInfo DBSettings) (string, error) {
	dbConnCmd, err := dbConnString(dbInfo)
	if err != nil {
		return "", err
	}

	loadCmd := fmt.Sprintf("zcat %s | %s", dumpFilename, dbConnCmd)

	return loadCmd, nil
}

// CmdLogger implements io.Writer to log command output through mlog
type CmdLogger struct{}

// Write logs the input to mlog and returns the length of the input
func (*CmdLogger) Write(in []byte) (int, error) {
	mlog.Info(strings.TrimSpace(string(in)))
	return len(in), nil
}

// RunCommand executes a command with proper logging
// If dst is set, it captures output to dst. Otherwise, it logs output through mlog.
func RunCommand(cmd *exec.Cmd, dst io.Writer) error {
	// If dst is set, that means we want to capture the output.
	// We write a simple case to handle that using CombinedOutput.
	if dst != nil {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		_, err = dst.Write(out)
		return err
	}

	cmd.Stdout = &CmdLogger{}
	cmd.Stderr = cmd.Stdout

	return cmd.Run()
}
