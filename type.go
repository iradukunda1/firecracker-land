package main

import (
	"context"
	"net/http"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
)

type CreateRequest struct {
	Name        string `json:"name" validate:"required"`
	DockerImage string `json:"docker-image" validate:"required"`
}

type CreateResponse struct {
	ID     string `json:"id,omitempty"`
	IpAddr string `json:"ip_address,omitempty"`
}

type DeleteRequest struct {
	ID string `json:"id" validate:"required"`
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

	IfName        string `long:"if-name" description:"if name to match your main ethernet adapter,the one that accesses the Internet - check 'ip addr' or 'ifconfig' if you don't know which one to use"`
	InitBaseTar   string `long:"init-base-tar" escription:"init-base-tar is our init base image file"` // make sure that this file is currently exists in the current directory by running task extract-init-base-tar
	ProvidedImage string `long:"provided-image" description:"provided-image is the image that we want to run in the VM"`
}

type Firecracker struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	machine   *firecracker.Machine
}

type Middleware func(h http.Handler) http.Handler
