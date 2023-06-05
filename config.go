package main

import (
	"fmt"
	"net"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func getOptions(id byte, req CreateRequest) options {
	fc_ip := net.IPv4(172, 102, 0, id).String()
	gateway_ip := "172.102.0.1"
	mask_long := "255.255.255.0" // reboot=k panic=1 pci=off init=/init ip=172.16.0.2::172.16.0.1:255.255.255.0::eth0:off root=/dev/vda"
	bootArgs := "ro console=ttyS0 noapic reboot=k panic=1 earlycon pci=off init=init nomodules random.trust_cpu=on "
	bootArgs = bootArgs + fmt.Sprintf("ip=%s::%s:%s::eth0:off", fc_ip, gateway_ip, mask_long)
	return options{
		VmIndex:        int64(id),
		FcBinary:       "firecracker",
		FcKernelImage:  "vmlinux.bin", // make sure that this file exists in the current directory with valid sum5
		KernelBootArgs: bootArgs,
		ProvidedImage:  req.DockerImage,
		ApiSocket:      fmt.Sprintf("/tmp/firecracker-ip%d.socket", id),
		TapMacAddr:     fmt.Sprintf("02:FC:00:00:00:%02x", id),
		Tap:            fmt.Sprintf("fc-tap-%d", id),
		// TapDev:     "tap0",
		// InitBaseTar: "rootfs.tar",
		FcIP:       fc_ip,
		BackBone:   "enp0s25", // eth0 or enp7s0,enp0s25
		FcCPUCount: 1,
		FcMemSz:    256,
	}
}

func (opts *options) getConfig() firecracker.Config {
	return firecracker.Config{
		VMID:            opts.Id,
		SocketPath:      opts.ApiSocket,
		KernelImagePath: opts.FcKernelImage,
		KernelArgs:      opts.KernelBootArgs,
		LogLevel:        "debug",
		InitrdPath:      "initrd.cpio",
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   &opts.RootFsImage,
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
				//Partuuid:     opts.FcRootPartUUID,
			},
		},

		//for setting up networking tap config vmmd config
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  opts.TapMacAddr,
					HostDevName: opts.Tap,
				},
				AllowMMDS: true,
			},
		},

		//for specifying the number of cpus and memory
		MachineCfg: models.MachineConfiguration{
			// VcpuCount:  &opts.FcCPUCount,
			// MemSizeMib: &opts.FcMemSz,
			// CPUTemplate: models.CPUTemplate(opts.FcCPUTemplate),
			// HtEnabled: firecracker.Bool(false),

			VcpuCount:  firecracker.Int64(1),
			Smt:        firecracker.Bool(false),
			MemSizeMib: firecracker.Int64(256),
		},

		// JailerCfg: jail,
		//VsockDevices:      vsocks,
		//LogFifo:           opts.FcLogFifo,
		//LogLevel:          opts.FcLogLevel,
		//MetricsFifo:       opts.FcMetricsFifo,
		//FifoLogWriter:     fifo,
	}
}
