// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

// Sync runs a terraform sync with all the required parameters.
func (t *Terraform) Sync() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	var params []string
	params = append(params, "refresh")
	params = append(params, t.getParams()...)
	params = append(params, "-state="+t.getStatePath())

	return t.runCommand(nil, params...)
}
