package main

import (
	"context"
	"fmt"
	"os"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/operations"
	log "github.com/sirupsen/logrus"
)

// Create a snapshot to a given path.
// Handles an existing VM socket path and a snapshot path.
func createSnapshot(socketPath string, snapshotPath string) error {
	cfg := firecracker.Config{SocketPath: socketPath}
	ctx := context.Background()

	// Create a logger to have a nice output
	logger := log.New()

	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithLogger(log.NewEntry(logger)))
	if err != nil {
		return fmt.Errorf("failed to create new machine: %v", err)
	}

	if err := machine.PauseVM(ctx); err != nil {
		return fmt.Errorf("failed to pause vm: %v", err)
	}

	start := time.Now()

	err = machine.CreateSnapshot(ctx, snapshotPath+".mem", snapshotPath+".file",
		func(data *operations.CreateSnapshotParams) {
			data.Body.SnapshotType = "Diff"
		})
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	fmt.Println("Created snapshot duration:", time.Since(start))

	if err := machine.ResumeVM(ctx); err != nil {
		return fmt.Errorf("failed to resume vm: %v", err)
	}

	return nil
}

// Load a snapshot from a given path.
// Handles VM socket path and a snapshot path.
func (o *options) loadSnapshot(ctx context.Context, snapshotPath string) error {

	if _, err := os.Stat(o.ApiSocket); err == nil {
		if err := os.Remove(o.ApiSocket); err != nil {
			return fmt.Errorf("failed to remove socket path: %v", err)
		}
	}

	cfg := firecracker.Config{
		SocketPath:        o.ApiSocket,
		DisableValidation: true,
		Snapshot: firecracker.SnapshotConfig{
			MemFilePath:         snapshotPath + ".mem",
			SnapshotPath:        snapshotPath + ".file",
			EnableDiffSnapshots: true,
			ResumeVM:            true,
		},
	}

	// Build the command
	cmd := firecracker.VMCommandBuilder{}.
		WithSocketPath(o.ApiSocket).
		WithBin(o.FcKernelImage).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)

	logger := log.New()

	// Start Firecracker
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start firecracker by resuming: %v", err)
	}
	defer cmd.Wait()

	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithLogger(log.NewEntry(logger)))
	if err != nil {
		return fmt.Errorf("failed to create new machine: %v", err)
	}

	// TODO: WaitForSocket interface could look better
	// errCh := make(chan error)
	if err := machine.Wait(ctx); err != nil {
		return fmt.Errorf("wait returned an error %v", err)
	}

	start := time.Now()

	if err := machine.ResumeVM(ctx); err != nil {
		return fmt.Errorf("failed to resume vm: %v", err)
	}

	// wait for the VMM to exit
	if err := machine.Wait(ctx); err != nil {
		return fmt.Errorf("wait returned an error %v", err)
	}

	fmt.Println("resuming snapshot duration:", time.Since(start))

	return nil
}
