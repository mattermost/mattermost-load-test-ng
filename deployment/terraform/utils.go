// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type uploadInfo struct {
	msg     string
	srcData string
	dstPath string
}

func uploadBatch(sshc *ssh.Client, batch []uploadInfo) error {
	if sshc == nil {
		return errors.New("sshc should not be nil")
	}
	if len(batch) == 0 {
		return errors.New("batch should not be empty")
	}

	for _, info := range batch {
		if info.msg != "" {
			mlog.Info(info.msg)
		}
		rdr := strings.NewReader(strings.TrimPrefix(info.srcData, "\n"))
		if out, err := sshc.Upload(rdr, info.dstPath, true); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", info.dstPath, out, err)
		}
	}

	return nil
}
