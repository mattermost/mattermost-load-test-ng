// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (t *Terraform) setupOpenLDAP(extAgent *ssh.ExtAgent) error {
	mlog.Info("Setting up OpenLDAP server")

	sshc, err := extAgent.NewClient(t.Config().AWSAMIUser, t.output.OpenLDAPServer.GetConnectionIP())
	if err != nil {
		return fmt.Errorf("error in getting ssh connection to OpenLDAP server %q: %w", t.output.OpenLDAPServer.GetConnectionIP(), err)
	}
	defer func() {
		err := sshc.Close()
		if err != nil {
			mlog.Error("error closing ssh connection", mlog.Err(err))
		}
	}()

	// Increase OpenLDAP database map size to handle large datasets
	// Default is usually 10MB which is too small for large LDIF imports
	ldifConfig := `dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcDbMaxSize
olcDbMaxSize: 4294967296` // 4GB. This is hardcoded. Needs to be coming from the instance spec.

	mlog.Info("Configuring OpenLDAP database size limits")

	// Upload the LDIF configuration
	if out, err := sshc.Upload(strings.NewReader(ldifConfig), "/tmp/increase_db_size.ldif", false); err != nil {
		return fmt.Errorf("error uploading OpenLDAP config LDIF: output: %s, error: %w", out, err)
	}

	// Apply the configuration
	cmd := "sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/increase_db_size.ldif"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error applying OpenLDAP config: command: %s, output: %s, error: %w", cmd, string(out), err)
	}

	// Clean up the temporary file
	cmd = "rm -f /tmp/increase_db_size.ldif"
	if out, err := sshc.RunCommand(cmd); err != nil {
		mlog.Error("error cleaning up temporary LDIF file", mlog.String("output", string(out)), mlog.Err(err))
	}

	mlog.Info("OpenLDAP database size configuration completed")
	return nil
}
