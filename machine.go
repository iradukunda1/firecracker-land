package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	llg "github.com/sirupsen/logrus"
)

// vmState this kind of vm-machine status
type VmState string

// avaliable vmState kind status
const (
	StateCreated VmState = "created"
	StateStarted VmState = "started"
	StateFailed  VmState = "failed"
	// StatePending VmState = "pending"
)

type Firecracker struct {
	Name      string
	ctx       context.Context
	cancelCtx context.CancelFunc
	vm        *firecracker.Machine
	state     VmState
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
	Logger        *llg.Logger
}

// JailingFirecrackerConfig represents Jailerspecific configuration options.
type JailingFirecrackerConfig struct {
	sync.Mutex

	BinaryFirecracker string `json:"BinaryFirecracker" mapstructure:"BinaryFirecracker"`
	BinaryJailer      string `json:"BinaryJailer" mapstructure:"BinaryJailer"`
	ChrootBase        string `json:"ChrootBase" mapstructure:"ChrootBase"`

	JailerGID      int `json:"JailerGid" mapstructure:"JailerGid"`
	JailerNumeNode int `json:"JailerNumaNode" mapstructure:"JailerNumaNode"`
	JailerUID      int `json:"JailerUid" mapstructure:"JailerUid"`

	NetNS string `json:"NetNS" mapstructure:"NetNS"`

	VmmID string
}

func installSignalHandlers(ctx context.Context, m *firecracker.Machine) {

	log := llg.New()

	go func() {
		// Clear some default handlers installed by the firecracker SDK:
		signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		for {
			switch s := <-c; {
			case s == syscall.SIGTERM || s == os.Interrupt:
				fmt.Println("Caught SIGTERM, requesting clean shutdown")
				if err := m.Shutdown(ctx); err != nil {
					log.Errorf("Machine shutdown failed with error: %v", err)
				}
				time.Sleep(20 * time.Second)

				// There's no direct way of checking if a VM is running, so we test if we can send it another shutdown
				// request. If that fails, the VM is still running and we need to kill it.
				if err := m.Shutdown(ctx); err == nil {
					fmt.Println("Timeout exceeded, forcing shutdown") // TODO: Proper logging
					if err := m.StopVMM(); err != nil {
						log.Errorf("VMM stop failed with error: %v", err)
					}
				}
			case s == syscall.SIGQUIT:
				fmt.Println("Caught SIGQUIT, forcing shutdown")
				if err := m.StopVMM(); err != nil {
					log.Errorf("VMM stop failed with error: %v", err)
				}
			}
		}
	}()
}

func Cleanup() {
	for _, run := range runVms {
		run.vm.StopVMM()
	}
}
