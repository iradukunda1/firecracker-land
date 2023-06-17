package main

import (
	"context"
	"fmt"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
)

var duration = 60 * time.Second

// CreateVmm is responsible to create vm and return its ip address
func (o *options) createVMM(ctx context.Context, id string) (*Firecracker, error) {

	vmCtx, _ := context.WithTimeout(ctx, duration)
	// defer vmCancel()

	llg := log.New()

	fcCfg := o.getConfig()

	// if fcCfg.JailerCfg == nil {
	// _ = firecracker.VMCommandBuilder{}.
	// 	WithBin(o.FcBinary).
	// 	WithSocketPath(fcCfg.SocketPath).
	// 	WithArgs([]string{"--id", id}).
	// 	WithStdin(os.Stdin).
	// 	WithStdout(os.Stdout).
	// 	WithStderr(os.Stderr).
	// 	Build(ctx)
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

	// g := errgroup.Group{}

	// out := make(chan *firecracker.Machine)

	// g.Go(func() error {

	m, err := firecracker.NewMachine(vmCtx, fcCfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating machine: %v", err)
	}

	// 	if err = machine.Start(vmCtx); err != nil {
	// 		return fmt.Errorf("failed to start machine: %v", err)
	// 	}

	// 	out <- machine

	// 	return nil
	// })

	// m := <-out

	// defer m.StopVMM()

	// if g.Wait() != nil {
	// 	log.Fatalf("error during start and create vm %v", g.Wait())
	// }

	// go func() {
	// 	m.Wait(ctx)
	// }()

	installSignalHandlers(vmCtx, m)

	// if err := m.Wait(ctx); err != nil {
	// 	log.Fatalf("failed to start machine during awaiting for vm health: %v", err)
	// }

	res := &Firecracker{
		ctx:  vmCtx,
		Name: o.ProvidedImage,
		// cancelCtx: vmCancel,
		vm:    m,
		state: StateCreated,
	}

	return res, nil
}
