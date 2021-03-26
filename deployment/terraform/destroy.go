// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	var params []string
	params = append(params, "destroy")
	params = append(params, t.getParams()...)
	params = append(params, "-auto-approve",
		"-input=false",
		"-state="+t.getStatePath(),
		t.dir)

	if err := t.runCommand(nil, params...); err != nil {
		return err
	}

	return t.loadOutput()
}
