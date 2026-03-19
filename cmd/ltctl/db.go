// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/spf13/cobra"
)

func RunDBListCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}

	if !output.HasDB() {
		return fmt.Errorf("no database cluster found in deployment")
	}

	readerIdx := 0
	for _, inst := range output.DBCluster.Instances {
		var role string
		if inst.IsWriter {
			role = "writer"
		} else {
			role = fmt.Sprintf("reader-%d", readerIdx)
			readerIdx++
		}
		fmt.Printf(" - %-10s %s  (%s)\n", role, inst.Endpoint, inst.DBIdentifier)
	}

	return nil
}

// selectDBInstance picks a DB instance based on the target argument.
// No args: first reader, or writer if only one instance exists.
// "writer": the writer instance.
// "reader-N": the reader at index N.
func selectDBInstance(output *terraform.Output, args []string) (terraform.DBInstance, error) {
	instances := output.DBCluster.Instances

	if len(args) == 0 {
		// Default: first reader, fallback to writer if only one instance.
		for _, inst := range instances {
			if !inst.IsWriter {
				return inst, nil
			}
		}
		// No readers found — return the writer.
		for _, inst := range instances {
			if inst.IsWriter {
				return inst, nil
			}
		}
		return terraform.DBInstance{}, fmt.Errorf("no database instances found")
	}

	target := args[0]

	if target == "writer" {
		for _, inst := range instances {
			if inst.IsWriter {
				return inst, nil
			}
		}
		return terraform.DBInstance{}, fmt.Errorf("no writer instance found")
	}

	if idxStr, ok := strings.CutPrefix(target, "reader-"); ok {
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return terraform.DBInstance{}, fmt.Errorf("invalid reader index %q", idxStr)
		}

		readerIdx := 0
		for _, inst := range instances {
			if !inst.IsWriter {
				if readerIdx == idx {
					return inst, nil
				}
				readerIdx++
			}
		}
		return terraform.DBInstance{}, fmt.Errorf("reader-%d not found (have %d readers)", idx, readerIdx)
	}

	return terraform.DBInstance{}, fmt.Errorf("invalid target %q: use \"writer\" or \"reader-N\"", target)
}

// selectJumpHost picks an EC2 instance to use as an SSH jump host.
// Fallback chain: app server -> metrics server -> error.
func selectJumpHost(output *terraform.Output) (terraform.Instance, error) {
	if output.HasAppServers() {
		return output.Instances[0], nil
	}
	if output.HasMetrics() {
		return output.MetricsServer, nil
	}
	return terraform.Instance{}, fmt.Errorf("no jump host available: need at least an app server or metrics server")
}

func RunDBConnectCmdF(cmd *cobra.Command, args []string) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return fmt.Errorf("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add")
	}

	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	// Validate engine is aurora-postgresql.
	if config.TerraformDBSettings.InstanceEngine != "aurora-postgresql" {
		return fmt.Errorf("only aurora-postgresql is supported, got %q", config.TerraformDBSettings.InstanceEngine)
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}

	if !output.HasDB() {
		return fmt.Errorf("no database cluster found in deployment")
	}

	// Select target DB instance and jump host.
	dbInst, err := selectDBInstance(output, args)
	if err != nil {
		return err
	}

	jumpHost, err := selectJumpHost(output)
	if err != nil {
		return err
	}

	// Establish SSH connection to the jump host.
	extAgent, err := ssh.NewAgent()
	if err != nil {
		return fmt.Errorf("failed to create SSH agent: %w", err)
	}

	sshClient, err := extAgent.NewClient(output.AMIUser, jumpHost.GetConnectionIP())
	if err != nil {
		return fmt.Errorf("failed to connect to jump host %s: %w", jumpHost.GetConnectionIP(), err)
	}
	defer sshClient.Close()

	fmt.Printf("Connected to jump host %s\n", jumpHost.GetConnectionIP())

	// Start local TCP listener on a free port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local listener: %w", err)
	}
	defer listener.Close()

	localAddr := listener.Addr().String()
	fmt.Printf("Tunnel listening on %s -> %s:5432\n", localAddr, dbInst.Endpoint)

	// Set up context with signal handling for clean shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		listener.Close()
	}()

	// Accept connections in the background and tunnel them via SSH.
	dialF := sshClient.DialContextF()
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				// Listener closed — exit goroutine.
				return
			}
			go func(lc net.Conn) {
				defer lc.Close()
				remoteConn, err := dialF(ctx, "tcp", dbInst.Endpoint+":5432")
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to dial remote DB: %v\n", err)
					return
				}
				defer remoteConn.Close()

				// Bidirectional copy.
				done := make(chan struct{}, 2)
				go func() {
					io.Copy(remoteConn, lc)
					done <- struct{}{}
				}()
				go func() {
					io.Copy(lc, remoteConn)
					done <- struct{}{}
				}()
				<-done
			}(localConn)
		}
	}()

	// Build psql connection string and exec.
	dbName := config.DBName()
	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		config.TerraformDBSettings.UserName,
		config.TerraformDBSettings.Password,
		localAddr,
		dbName,
	)

	fmt.Printf("Connecting to %s on %s...\n", dbName, dbInst.Endpoint)

	psql := exec.CommandContext(ctx, "psql", connStr)
	psql.Stdin = os.Stdin
	psql.Stdout = os.Stdout
	psql.Stderr = os.Stderr

	if err := psql.Run(); err != nil {
		// If killed by our signal handler, don't treat as error.
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("psql exited with error: %w", err)
	}

	return nil
}
