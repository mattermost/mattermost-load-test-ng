// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
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

// ProvisionURL takes a URL pointing to a file to be provisioned.
// It works on both local files prefixed with file:// or remote files.
// In case of local files, they are uploaded to the server.
func ProvisionURL(client *ssh.Client, url, filename string) error {
	filePrefix := "file://"
	if strings.HasPrefix(url, filePrefix) {
		// upload file from local filesystem
		path := strings.TrimPrefix(url, filePrefix)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("build file %s has to be a regular file", path)
		}
		if out, err := client.UploadFile(path, fmt.Sprintf("$HOME/%s", filename), false); err != nil {
			return fmt.Errorf("error uploading build: %w %s", err, out)
		}
	} else {
		// download build file from URL
		cmd := fmt.Sprintf("wget -O %s %s", filename, url)
		if out, err := client.RunCommand(cmd); err != nil {
			return fmt.Errorf("failed to run cmd %q: %w %s", cmd, err, out)
		}
	}

	return nil
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
