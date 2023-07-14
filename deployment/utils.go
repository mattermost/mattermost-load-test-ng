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
		if out, err := client.UploadFile(path, "/home/ubuntu/"+filename, false); err != nil {
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

// BuildLoadDBDumpCmds returns a slice of commands that, when piped, feed the
// provided DB dump file into the database, replacing first the old IPs found
// in the posts that contain a permalink with the new IP. Something like:
//
//	zcat dbdump.sql
//	sed -r -e 's/old_ip_1/new_ip' -e 's/old_ip_2/new_ip'
//	mysql/psql connection_details
func BuildLoadDBDumpCmds(dumpFilename string, newIP string, permalinkIPsToReplace []string, dbInfo DBSettings) ([]string, error) {
	zcatCmd := fmt.Sprintf("zcat %s", dumpFilename)

	var replacements []string
	for _, oldIP := range permalinkIPsToReplace {
		// Let's build the match and replace parts of a sed command: 's/match/replace/g'
		// First, the match. We want to match anything of the form
		//    54.126.54.26:8065/debitis-1/pl/
		// where the IP is exactly the old one, the port is optional and arbitrary and the
		// team name is the pattern defined by the server's function model.IsValidTeamname
		validTeamName := `[a-z0-9]+([a-z0-9-]+|(__)?)[a-z0-9]+`
		escapedOldIP := strings.ReplaceAll(oldIP, ".", "\\.")
		match := escapedOldIP + `(:[0-9]+)?\/(` + validTeamName + `)\/pl\/`
		// Now, the replace. We need to replace this with the same thing, only changing the
		// IP with the new one and hard-coding the port to 8065, but maintaining the team
		// name (hence the second group match, \2)
		replace := newIP + `:8065\/\2\/pl\/`
		// We can build the whole command now and add it to the list of replacements
		sedRegex := fmt.Sprintf(`'s/%s/%s/g'`, match, replace)
		replacements = append(replacements, sedRegex)
	}
	var sedCmd string
	if len(replacements) > 0 {
		sedCmd = strings.Join(append([]string{"sed -r"}, replacements...), " -e ")
	}

	var dbCmd string
	switch dbInfo.Engine {
	case "aurora-postgresql":
		dbCmd = fmt.Sprintf("psql 'postgres://%[1]s:%[2]s@%[3]s/%[4]s?sslmode=disable'", dbInfo.UserName, dbInfo.Password, dbInfo.Host, dbInfo.DBName)
	case "aurora-mysql":
		dbCmd = fmt.Sprintf("mysql -h %[1]s -u %[2]s -p%[3]s %[4]s", dbInfo.Host, dbInfo.UserName, dbInfo.Password, dbInfo.DBName)
	default:
		return []string{}, fmt.Errorf("invalid db engine %s", dbInfo.Engine)
	}

	if sedCmd != "" {
		return []string{zcatCmd, sedCmd, dbCmd}, nil
	}

	return []string{zcatCmd, dbCmd}, nil
}
