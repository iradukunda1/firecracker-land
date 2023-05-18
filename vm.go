package main

import (
	"context"
	"fmt"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/sirupsen/logrus"
)

// StoppedOK is the VMM stopped status.
type StoppedOK = bool

var (
	// StoppedGracefully indicates the machine was stopped gracefully.
	StoppedGracefully = StoppedOK(true)
	// StoppedForcefully indicates that the machine did not stop gracefully
	// and the shutdown had to be forced.
	StoppedForcefully = StoppedOK(false)
)

// Provider abstracts the configuration required to start a VMM.
type Provider interface {
	// Start starts the VMM.
	Start(context.Context) (StartedMachine, error)

	WithHandlersAdapter(firecracker.HandlersAdapter) Provider
	WithVethIfaceName(string) Provider
}

type defaultProvider struct {
	// cniConfig       *configs.CNIConfig
	jailingFcConfig *JailingFirecrackerConfig
	machineConfig   *MachineCfg

	handlersAdapter firecracker.HandlersAdapter
	logger          logrus.Logger
	machine         *firecracker.Machine
	vethIfaceName   string
}

// NewDefaultProvider creates a default provider.
func NewDefaultProvider(machine *MachineCfg, jail *JailingFirecrackerConfig) Provider {
	return &defaultProvider{
		// cniConfig:       cniConfig,
		jailingFcConfig: jail,
		machineConfig:   machine,

		handlersAdapter: DefaultFirectackerStrategy(machine),
		logger:          *logrus.New(),
		vethIfaceName:   DefaultVethIfaceName,
	}
}

func (p *defaultProvider) Start(ctx context.Context) (StartedMachine, error) {

	machineChroot := NewWithLocation(LocationFromComponents(p.jailingFcConfig.JailerChrootDirectory(),
		p.jailingFcConfig.BinaryFirecracker,
		p.jailingFcConfig.VMMID()))

	vmmLoggerEntry := logrus.NewEntry(logrus.New())
	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(vmmLoggerEntry),
	}

	if p.machineConfig.LogFcHTTPCalls {
		machineOpts = append(machineOpts, firecracker.
			WithClient(firecracker.NewClient(machineChroot.SocketPath(), vmmLoggerEntry, true)))
	}

	fcConfig := NewFcConfigProvider(p.jailingFcConfig, p.machineConfig).
		WithHandlersAdapter(p.handlersAdapter).
		WithVethIfaceName(p.vethIfaceName).
		ToSDKConfig()

	m, err := firecracker.NewMachine(ctx, fcConfig, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed creating machine: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		return nil, fmt.Errorf("Failed to start machine: %v", err)
	}

	return &defaultStartedMachine{
		// cniConfig:       p.cniConfig,
		jailingFcConfig: p.jailingFcConfig,
		machineConfig:   p.machineConfig,
		logger:          p.logger,
		machine:         m,
		vethIfaceName:   p.vethIfaceName,
	}, nil
}

func (p *defaultProvider) WithHandlersAdapter(input firecracker.HandlersAdapter) Provider {
	p.handlersAdapter = input
	return p
}

func (p *defaultProvider) WithVethIfaceName(input string) Provider {
	p.vethIfaceName = input
	return p
}
