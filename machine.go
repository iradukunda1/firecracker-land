package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
)

type Firecracker struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	machine   *firecracker.Machine
	Agent     net.IP
}

type options struct {
	Id             string `long:"id" description:"Jailer VMM id"`
	VmIndex        int64  `long:"vm-index" description:"VM index"`
	ApiSocket      string `long:"socket-path" short:"s" description:"path to use for firecracker socket"`
	IpId           byte   `byte:"id" description:"an ip we use to generate an ip address"`
	FcBinary       string `long:"firecracker-binary" description:"Path to firecracker binary"`
	FcKernelImage  string `long:"kernel" description:"Path to the kernel image"`
	KernelBootArgs string `long:"kernel-opts" description:"Kernel commandline"`
	RootFsImage    string `long:"root-drive" description:"Path to root disk image"`
	TapMacAddr     string `long:"tap-mac-addr" description:"tap macaddress"`
	Tap            string `long:"tap-dev" description:"tap device"`
	FcCPUCount     int64  `long:"ncpus" short:"c" description:"Number of CPUs"`
	FcMemSz        int64  `long:"memory" short:"m" description:"VM memory, in MiB"`
	FcIP           string `long:"fc-ip" description:"IP address of the VM"`

	BackBone      string `long:"if-name" description:"if name to match your main ethernet adapter,the one that accesses the Internet - check 'ip addr' or 'ifconfig' if you don't know which one to use"` // eg eth0
	InitBaseTar   string `long:"init-base-tar" description:"init-base-tar is our init base image file"`                                                                                                   // make sure that this file is currently exists in the current directory by running task extract-init-base-tar
	ProvidedImage string `long:"provided-image" description:"provided-image is the image that we want to run in the VM"`
	InitdPath     string `long:"initd-path" description:"initd-path is the path to the init binary file"`
}

func installSignalHandlers(ctx context.Context, m *firecracker.Machine) {

	lg := log.New()

	// not sure if this is actually really helping with anything
	go func() {
		// Clear some default handlers installed by the firecracker SDK:
		signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		for {
			switch s := <-c; {
			case s == syscall.SIGTERM || s == os.Interrupt:
				lg.Printf("Caught SIGINT, requesting clean shutdown")
				m.Shutdown(ctx)
			case s == syscall.SIGQUIT:
				lg.Printf("Caught SIGTERM, forcing shutdown")
				m.StopVMM()
			}
		}
	}()
}

func Cleanup() {
	for _, run := range runVms {
		run.machine.StopVMM()
	}
}
