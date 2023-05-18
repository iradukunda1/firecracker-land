package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

// DefaultVethIfaceName is the default veth interface name.
const DefaultVethIfaceName = "veth0"

type defaultFcConfigProvider struct {
	jailingFcConfig      *JailingFirecrackerConfig
	machineCfgMachineCfg *MachineCfg

	fcStrategy    firecracker.HandlersAdapter
	vethIfaceName string
}

// FcConfigProvider is a Firecracker SDK configuration builder provider.
type FcConfigProvider interface {
	ToSDKConfig() firecracker.Config
	WithHandlersAdapter(firecracker.HandlersAdapter) FcConfigProvider
	WithVethIfaceName(string) FcConfigProvider
}

// DefaultFirectackerStrategy returns an instance of the default Firecracker Jailer strategy for a given machine config.
func DefaultFirectackerStrategy(cfg *MachineCfg) PlacingStrategy {
	return NewStrategy(func() *HandlerPlacement {
		return NewHandlerPlacement(firecracker.
			LinkFilesHandler(filepath.Base(cfg.KernelOverride())),
			firecracker.CreateLogFilesHandlerName)
	})
}

// JailerChrootDirectory returns a full path to the jailer configuration directory.
// This method will return empty string until the flag set returned by FlagSet() is parsed.
func (c *JailingFirecrackerConfig) JailerChrootDirectory() string {
	return filepath.Join(c.ChrootBase,
		filepath.Base(c.BinaryFirecracker), c.VMMID())
}

// VMMID returns a configuration instance unique VMM ID.
func (c *JailingFirecrackerConfig) VMMID() string {
	return c.vmmID
}

// NewFcConfigProvider creates a new builder provider.
func NewFcConfigProvider(jailingFcConfig *JailingFirecrackerConfig, machineCfgMachineCfg *MachineCfg) FcConfigProvider {
	return &defaultFcConfigProvider{
		jailingFcConfig:      jailingFcConfig,
		machineCfgMachineCfg: machineCfgMachineCfg,
		vethIfaceName:        DefaultVethIfaceName,
	}
}

func (c *defaultFcConfigProvider) ToSDKConfig() firecracker.Config {

	var fifo io.WriteCloser // CONSIDER: do it like firectl does it

	return firecracker.Config{
		SocketPath:      "",      // given via Jailer
		LogFifo:         "",      // CONSIDER: make this configurable
		LogLevel:        "debug", // CONSIDER: make this configurable
		MetricsFifo:     "",      // not configurable for the build machines
		FifoLogWriter:   fifo,
		KernelImagePath: c.machineCfgMachineCfg.KernelOverride(),
		KernelArgs:      c.machineCfgMachineCfg.KernelArgs,
		NetNS:           c.jailingFcConfig.NetNS,
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   firecracker.String(c.machineCfgMachineCfg.RootfsOverride()),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
				Partuuid:     c.machineCfgMachineCfg.RootDrivePartUUID,
			},
		},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			AllowMMDS: !c.machineCfgMachineCfg.NoMMDS,
			CNIConfiguration: &firecracker.CNIConfiguration{
				NetworkName: c.machineCfgMachineCfg.CNINetworkName,
				IfName:      c.vethIfaceName,
				Args: func() [][2]string {
					if c.machineCfgMachineCfg.IPAddress != "" {
						return [][2]string{
							{"IP", c.machineCfgMachineCfg.IPAddress},
						}
					}
					return [][2]string{}
				}(),
			},
		}},
		VsockDevices: []firecracker.VsockDevice{},

		MachineCfg: models.MachineConfiguration{
			VcpuCount:   firecracker.Int64(c.machineCfgMachineCfg.CPU),
			CPUTemplate: models.CPUTemplate(c.machineCfgMachineCfg.CPUTemplate),
			// HtEnabled:   firecracker.Bool(c.machineCfgMachineCfg.HTEnabled),
			MemSizeMib: firecracker.Int64(c.machineCfgMachineCfg.Mem),
		},

		JailerCfg: &firecracker.JailerConfig{
			GID:           firecracker.Int(c.jailingFcConfig.JailerGID),
			UID:           firecracker.Int(c.jailingFcConfig.JailerUID),
			ID:            c.jailingFcConfig.VMMID(),
			NumaNode:      firecracker.Int(c.jailingFcConfig.JailerNumeNode),
			ExecFile:      c.jailingFcConfig.BinaryFirecracker,
			JailerBinary:  c.jailingFcConfig.BinaryJailer,
			ChrootBaseDir: c.jailingFcConfig.ChrootBase,
			Daemonize:     c.machineCfgMachineCfg.Daemonize(),
			ChrootStrategy: func() firecracker.HandlersAdapter {
				if c.fcStrategy == nil {
					return DefaultFirectackerStrategy(c.machineCfgMachineCfg)
				}
				return c.fcStrategy
			}(),
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			// do not pass stdin because the build VMM does not require input
			// and it messes up the terminal
			Stdin: nil,
		},
		VMID: c.jailingFcConfig.VMMID(),
	}
}

func (c *defaultFcConfigProvider) WithHandlersAdapter(input firecracker.HandlersAdapter) FcConfigProvider {
	c.fcStrategy = input
	return c
}

func (c *defaultFcConfigProvider) WithVethIfaceName(input string) FcConfigProvider {
	c.vethIfaceName = input
	return c
}

// Daemonize returns the configured daemonize setting.
func (c *MachineCfg) Daemonize() bool {
	return c.daemonize
}

// KernelOverride returns the configured kernel setting.
func (c *MachineCfg) KernelOverride() string {
	return c.kernelOverride
}

// RootfsOverride returns the configured rootfs setting.
func (c *MachineCfg) RootfsOverride() string {
	return c.rootfsOverride
}

// WithDaemonize sets the daemonize setting.
func (c *MachineCfg) WithDaemonize(input bool) *MachineCfg {
	c.daemonize = input
	return c
}

// WithKernelOverride sets the ketting setting.
func (c *MachineCfg) WithKernelOverride(input string) *MachineCfg {
	c.kernelOverride = input
	return c
}

// WithRootfsOverride sets the rootfs setting.
func (c *MachineCfg) WithRootfsOverride(input string) *MachineCfg {
	c.rootfsOverride = input
	return c
}
