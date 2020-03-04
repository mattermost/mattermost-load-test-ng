// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	err := t.preFlightCheck()
	if err != nil {
		return err
	}

	err = t.runCommand(nil, "destroy",
		"-auto-approve",
		"./terraform",
	)
	if err != nil {
		return err
	}
	return nil
}
