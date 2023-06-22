package main

import (
	"context"
	"fmt"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
)

var duration = 120 * time.Second

// CreateVmm is responsible to create vm and return its ip address
func (o *options) createVMM(ctx context.Context, id string) (*Firecracker, error) {

	llg := log.New()

	cfg := o.getConfig()

	// client := firecracker.NewClient(fcCfg.SocketPath, llg.WithContext(vmmCtx), true)

	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(log.NewEntry(llg)),
	}

	cfg.VMID = id
	cfg.JailerCfg.ID = id

	if err := exposeBlockDeviceToJail(o.RootFsImage, *cfg.JailerCfg.UID, *cfg.JailerCfg.GID); err != nil {
		return nil, fmt.Errorf("failed to expose fs to jail: %v", err)
	}

	// remove old socket path if it exists
	if _, err := RunNoneSudo(fmt.Sprintf("rm -f %s > /dev/null || true", o.ApiSocket)); err != nil {
		return nil, fmt.Errorf("failed to delete old socket path: %s", err)
	}

	if err := o.SetNetwork(); err != nil {
		return nil, fmt.Errorf("failed to set network: %s", err)
	}

	m, err := firecracker.NewMachine(ctx, cfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating machine: %v", err)
	}

	installSignalHandlers(ctx, m)

	res := &Firecracker{
		ctx:  ctx,
		Name: o.ProvidedImage,
		// cancelCtx: nil,
		vm:    m,
		state: StateCreated,
	}

	return res, nil
}
