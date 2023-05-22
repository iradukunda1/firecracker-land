package main

import (
	"fmt"
	"net"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func getOptions(id byte, req CreateRequest) options {
	fc_ip := net.IPv4(172, 17, 0, id).String()
	gateway_ip := "172.17.0.1"
	docker_mask_long := "255.255.255.0"
	bootArgs := "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules random.trust_cpu=on "
	bootArgs = bootArgs + fmt.Sprintf("ip=%s::%s:%s::eth0:off", fc_ip, gateway_ip, docker_mask_long)
	return options{
		VmIndex:        int64(id),
		FcBinary:       "firecracker",
		FcKernelImage:  "vmlinux.bin", // make sure that this file exists in the current directory with valid sum5
		KernelBootArgs: bootArgs,
		ProvidedImage:  req.DockerImage,
		ApiSocket:      fmt.Sprintf("/tmp/firecracker-ip%d.sock", id),
		TapMacAddr:     fmt.Sprintf("02:FC:00:00:00:%02x", id),
		Tap:            fmt.Sprintf("fc-tap-%d", id),
		// TapDev:     "tap0",
		InitBaseTar: "rootfs.tar",
		FcIP:        fc_ip,
		IfName:      "enp7s0", // eth0
		FcCPUCount:  1,
		FcMemSz:     512,
	}
}

func (opts *options) getConfig() firecracker.Config {
	return firecracker.Config{
		VMID:            opts.Id,
		SocketPath:      opts.ApiSocket,
		KernelImagePath: opts.FcKernelImage,
		KernelArgs:      opts.KernelBootArgs,
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   &opts.RootFsImage,
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
				//Partuuid:     opts.FcRootPartUUID,
			},
		},

		//for setting up networking tap config
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  opts.TapMacAddr,
					HostDevName: opts.Tap,
				},
				// AllowMMDS: true,
			},
		},

		//for specifying the number of cpus and memory
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(opts.FcCPUCount),
			MemSizeMib: firecracker.Int64(opts.FcMemSz),
			//CPUTemplate: models.CPUTemplate(opts.FcCPUTemplate),
			// HtEnabled: firecracker.Bool(false),
		},
		// JailerCfg: jail,
		//VsockDevices:      vsocks,
		//LogFifo:           opts.FcLogFifo,
		//LogLevel:          opts.FcLogLevel,
		//MetricsFifo:       opts.FcMetricsFifo,
		//FifoLogWriter:     fifo,
	}
}
