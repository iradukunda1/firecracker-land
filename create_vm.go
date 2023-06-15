package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// CreateVmm is responsible to create vm and return its ip address
func (o *options) createVMM(ctx context.Context, id string) (*Firecracker, error) {

	vmCtx, vmCancel := context.WithCancel(ctx)
	defer vmCancel()

	llg := log.New()

	fcCfg := o.getConfig()

	// if fcCfg.JailerCfg == nil {
	_ = firecracker.VMCommandBuilder{}.
		WithBin(o.FcBinary).
		WithSocketPath(fcCfg.SocketPath).
		WithArgs([]string{"--id", id}).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)
	// }

	// client := firecracker.NewClient(fcCfg.SocketPath, llg.WithContext(vmmCtx), true)

	machineOpts := []firecracker.Opt{
		// firecracker.WithClient(client),
		// firecracker.WithProcessRunner(cmd),
		// firecracker.WithProcessRunner(o.jailerCommand(vmmCtx, id, true)),
		firecracker.WithLogger(log.NewEntry(llg)),
	}

	fcCfg.VMID = id
	fcCfg.JailerCfg.ID = id
	if err := exposeBlockDeviceToJail(o.RootFsImage, *fcCfg.JailerCfg.UID, *fcCfg.JailerCfg.GID); err != nil {
		return nil, fmt.Errorf("failed to expose fs to jail: %v", err)
	}

	// remove old socket path if it exists
	if _, err := RunNoneSudo(fmt.Sprintf("rm -f %s > /dev/null || true", o.ApiSocket)); err != nil {
		return nil, fmt.Errorf("failed to delete old socket path: %s", err)
	}

	if err := o.SetNetwork(); err != nil {
		return nil, fmt.Errorf("failed to set network: %s", err)
	}

	m, err := firecracker.NewMachine(vmCtx, fcCfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating machine: %s", err)
	}
	defer m.StopVMM()

	g := errgroup.Group{}

	g.Go(func() error {
		if err = m.Start(vmCtx); err != nil {
			return fmt.Errorf("failed to start machine: %v", err)
		}
		return nil
	})
	defer m.StopVMM()

	installSignalHandlers(vmCtx, m)

	if err := m.Wait(ctx); err != nil {
		log.Fatalf("failed to start machine during awaiting for vm health: %v", err)
	}

	return &Firecracker{
		ctx:       vmCtx,
		cancelCtx: vmCancel,
		machine:   m,
	}, nil
}

func (opt *options) jailerCommand(ctx context.Context, containerName string, isDebug bool) *exec.Cmd {

	fc := opt.getConfig()

	cmd := exec.CommandContext(ctx, fc.JailerCfg.JailerBinary, "run", containerName)
	cmd.Dir = fc.JailerCfg.ChrootBaseDir

	if isDebug {
		cmd.Stdout = opt.Logger.WithField("vmm_stream", "stdout").WriterLevel(log.DebugLevel)
		cmd.Stderr = opt.Logger.WithField("vmm_stream", "stderr").WriterLevel(log.DebugLevel)
	}

	return cmd
}
