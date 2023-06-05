package main

import (
	"context"
	"fmt"
	"os"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// CreateVmm is responsible to create vm and return its ip address
func (o *options) createVMM(ctx context.Context, id string) (*Firecracker, error) {

	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	fcCfg := o.getConfig()

	cmd := firecracker.VMCommandBuilder{}.
		WithBin(o.FcBinary).
		WithSocketPath(fcCfg.SocketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)
	machineOpts := []firecracker.Opt{
		firecracker.WithProcessRunner(cmd),
		firecracker.WithLogger(log.NewEntry(log.New())),
	}

	fcCfg.VMID = id

	// remove old socket path if it exists
	if _, err := RunNoneSudo(fmt.Sprintf("rm -f %s", o.ApiSocket)); err != nil {
		return nil, fmt.Errorf("failed to delete old socket path: %s", err)
	}

	if err := o.SetNetwork(); err != nil {
		return nil, fmt.Errorf("failed to set network: %s", err)
	}

	m, err := firecracker.NewMachine(vmmCtx, fcCfg, machineOpts...)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("failed creating machine: %s", err)
	}
	defer m.StopVMM()

	g := errgroup.Group{}

	g.Go(func() error {
		if err = m.Start(vmmCtx); err != nil {
			return fmt.Errorf("failed to start machine: %v", err)
		}
		return nil
	})
	defer m.StopVMM()

	if err := m.Wait(ctx); err != nil {
		log.Fatalf("failed to start machine during awaiting for vm health: %v", err)
	}

	installSignalHandlers(vmmCtx, m)

	return &Firecracker{
		ctx:       vmmCtx,
		cancelCtx: vmmCancel,
		machine:   m,
	}, nil
}
