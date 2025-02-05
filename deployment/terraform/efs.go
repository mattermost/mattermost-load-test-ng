package terraform

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	efsDirectory  = "/mnt/mattermost-data"
	svcScript     = "/opt/mount.sh"
	efcServiceDir = "/lib/systemd/system/efs-mount.service"
	efsService    = `[Unit]
Description=Script to run after fstab to change ownership of mount directory
After=local-fs.target

[Service]
Type=simple
ExecStartPre=chown -R ubuntu:ubuntu /mnt/mattermost-data
ExecStart=/bin/bash -c "/opt/mount.sh"

[Install]
WantedBy=multi-user.target
`
)

func (t *Terraform) setupEFS(extAgent *ssh.ExtAgent) error {
	for _, val := range t.output.Instances {
		sshc, err := extAgent.NewClient(val.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer sshc.Close()

		cmd := fmt.Sprintf("sudo mkdir -p %s", efsDirectory)
		_, err = sshc.RunCommand(cmd)
		if err != nil {
			return fmt.Errorf("error creating the mounting point: %w", err)
		}

		mlog.Info("Updating /etc/fstab to mount EFS", mlog.String("fsid", t.output.EFSAccessPoint.FileSystemId))
		// setting uid and gid to ubuntu user
		fstabEntry := fmt.Sprintf("%s.efs.%s.amazonaws.com:/ %s nfs4 nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2,noresvport,_netdev 0 0",
			t.output.EFSAccessPoint.FileSystemId,
			t.config.AWSRegion,
			efsDirectory)
		cmd = fmt.Sprintf("grep -q %q /etc/fstab || sudo sh -c 'echo %q >> /etc/fstab'", efsDirectory, fstabEntry)
		_, err = sshc.RunCommand(cmd)
		if err != nil {
			return fmt.Errorf("error modifying /etc/fstab: %w", err)
		}

		_, err = sshc.RunCommand("sudo mount -a")
		if err != nil {
			return fmt.Errorf("error mounting the fstab entry: %w", err)
		}

		rdr := strings.NewReader(fmt.Sprintf("sudo chown -R ubuntu:ubuntu %s", efsDirectory))
		if out, err := sshc.Upload(rdr, svcScript, true); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", efcServiceDir, out, err)
		}

		cmd = fmt.Sprintf("sudo chmod +x %s", svcScript)
		_, err = sshc.RunCommand(cmd)
		if err != nil {
			return fmt.Errorf("error creating the script to change ownership: %w", err)
		}

		rdr = strings.NewReader(efsService)
		if out, err := sshc.Upload(rdr, efcServiceDir, true); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", efcServiceDir, out, err)
		}

		cmd = "sudo service efs-mount start"
		_, err = sshc.RunCommand(cmd)
		if err != nil {
			return fmt.Errorf("error starting the service for efs mount directory ownership: %w", err)
		}
	}

	return nil
}
