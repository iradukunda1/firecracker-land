package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/iradukunda1/utils"
	"golang.org/x/sync/errgroup"
)

type CreateRequest struct {
	RootDrivePath string `json:"root_image_path"`
	KernelPath    string `json:"kernel_path"`
}

type CreateResponse struct {
	IpAddress string `json:"ip_address"`
	ID        string `json:"id"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

var runningVMs map[string]RunningFirecracker = make(map[string]RunningFirecracker)
var ipByte byte = 3

func main() {

	http.HandleFunc("/create", createRequestHandler)
	http.HandleFunc("/delete", deleteRequestHandler)
	defer cleanup()

	log.Println("Listening on port 8080")

	ctx, cancel := context.WithCancel(context.Background())

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			cancel()
		case <-ctx.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(1)
	}()

	g := errgroup.Group{}

	g.Go(func() error {
		return http.ListenAndServe(":8080", nil)
	})

	<-ctx.Done()

	log.Println("shutting down app")

	if err := g.Wait(); err != nil {
		log.Fatal("main: runtime program terminated")
	}
}

func cleanup() {
	for _, running := range runningVMs {
		running.machine.StopVMM()
	}
}

func deleteRequestHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	var req DeleteRequest
	json.Unmarshal([]byte(body), &req)
	if err != nil {
		log.Fatalf(err.Error())
	}

	running := runningVMs[req.ID]
	running.machine.StopVMM()
	delete(runningVMs, req.ID)
}

func createRequestHandler(w http.ResponseWriter, r *http.Request) {
	ipByte += 1
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	var req CreateRequest
	json.Unmarshal([]byte(body), &req)
	opts := getOptions(ipByte, req)
	running, err := opts.createVMM(context.Background())
	if err != nil {
		log.Fatalf(err.Error())
	}

	id := pseudo_uuid()
	resp := CreateResponse{
		IpAddress: opts.FcIP,
		ID:        id,
	}
	response, err := json.Marshal(&resp)
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(response)

	runningVMs[id] = *running

	go func() {
		defer running.cancelCtx()
		// there's an error here but we ignore it for now because we terminate
		// the VM on /delete and it returns an error when it's terminated
		running.machine.Wait(running.ctx)
	}()
}

func pseudo_uuid() string {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("failed to generate uuid, %s", err)
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func getOptions(id byte, req CreateRequest) options {
	fc_ip := net.IPv4(172, 17, 0, id).String()
	gateway_ip := "172.17.0.1"
	docker_mask_long := "255.255.255.0"
	bootArgs := "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules random.trust_cpu=on "
	bootArgs = bootArgs + fmt.Sprintf("ip=%s::%s:%s::eth0:off", fc_ip, gateway_ip, docker_mask_long)
	return options{
		FcBinary:         "/home/dedsec/firecracker",
		FcKernelImage:    req.KernelPath,
		FckernelBootArgs: bootArgs,
		FcRootDrivePath:  req.RootDrivePath,
		FcSocketPath:     fmt.Sprintf("/tmp/firecracker-ip%d.sock", id),
		TapMacAddr:       fmt.Sprintf("02:FC:00:00:00:%02x", id),
		TapDev:           fmt.Sprintf("fc-tap-%d", id),
		// TapDev:     "tap0",
		FcIP:       fc_ip,
		IfName:     "enp7s0", // eth0
		FcCPUCount: 1,
		FcMemSz:    512,
	}
}

type RunningFirecracker struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	machine   *firecracker.Machine
}

func (opts *options) createVMM(ctx context.Context) (*RunningFirecracker, error) {
	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	fcCfg := opts.getConfig()

	cmd := firecracker.VMCommandBuilder{}.
		WithBin(opts.FcBinary).
		WithSocketPath(fcCfg.SocketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)

	machineOpts := []firecracker.Opt{
		firecracker.WithProcessRunner(cmd),
	}

	// remove old socket path if it exists
	if _, err := utils.RunShellCommandNoSudo(fmt.Sprintf("rm -f %s", opts.FcSocketPath)); err != nil {
		return nil, fmt.Errorf("failed to delete old socket path: %s", err)
	}

	// delete tap device if it exists
	if res, err := utils.RunShellCommandSudo(fmt.Sprintf("ip link del %s", opts.TapDev)); res != 1 && err != nil {
		return nil, fmt.Errorf("failed during deleting tap device: %v", err)
	}

	// create tap device
	if _, err := utils.RunShellCommandSudo(fmt.Sprintf("ip tuntap add dev %s mode tap", opts.TapDev)); err != nil {
		return nil, fmt.Errorf("failed creating ip link for tap: %s", err)
	}

	// set tap device mac address
	if _, err := utils.RunShellCommandSudo(fmt.Sprintf("ip addr add %s/24 dev %s", opts.FcIP, opts.TapDev)); err != nil {
		return nil, fmt.Errorf("failed to add ip address on tap device: %v", err)
	}

	// set tap device up by activating it
	if _, err := utils.RunShellCommandSudo(fmt.Sprintf("ip link set %s up", opts.TapDev)); err != nil {
		return nil, fmt.Errorf("failed to set tap device up: %v", err)
	}

	// show tap connection created
	if _, err := utils.RunShellCommandNoSudo(fmt.Sprintf("ip addr show dev %s", opts.TapDev)); err != nil {
		return nil, fmt.Errorf("failed to show tap connection created: %v", err)
	}

	//enable ip forwarding
	if _, err := utils.RunShellCommandSudo(" sh -c 'echo 1 > /proc/sys/net/ipv4/ip_forward'"); err != nil {
		return nil, fmt.Errorf("failed to enable ip forwarding: %v", err)
	}

	// add iptables rule to forward packets from tap to eth0
	if _, err := utils.RunShellCommandSudo(fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", opts.IfName)); err != nil {
		return nil, fmt.Errorf("failed to add iptables rule to forward packets from tap to eth0: %v", err)
	}

	// add iptables rule to establish connection between tap and eth0 (forward packets from eth0 to tap)
	if _, err := utils.RunShellCommandSudo("iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT"); err != nil {
		return nil, fmt.Errorf("failed to add iptables rule to establish connection between tap and eth0: %v", err)
	}

	// add iptables rule to forward packets from eth0 to tap
	if _, err := utils.RunShellCommandSudo(fmt.Sprintf("iptables -A FORWARD -i %s -o %s -j ACCEPT", opts.IfName, opts.TapDev)); err != nil {
		return nil, fmt.Errorf("failed to add iptables rule to forward packets from eth0 to tap: %v", err)
	}

	m, err := firecracker.NewMachine(vmmCtx, fcCfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating machine: %s", err)
	}
	if err := m.Start(vmmCtx); err != nil {
		return nil, fmt.Errorf("failed to start machine: %v", err)
	}
	installSignalHandlers(vmmCtx, m)
	return &RunningFirecracker{
		ctx:       vmmCtx,
		cancelCtx: vmmCancel,
		machine:   m,
	}, nil
}

type options struct {
	Id string `long:"id" description:"Jailer VMM id"`
	// maybe make this an int instead
	IpId             byte   `byte:"id" description:"an ip we use to generate an ip address"`
	FcBinary         string `long:"firecracker-binary" description:"Path to firecracker binary"`
	FcKernelImage    string `long:"kernel" description:"Path to the kernel image"`
	FckernelBootArgs string `long:"kernel-opts" description:"Kernel commandline"`
	FcRootDrivePath  string `long:"root-drive" description:"Path to root disk image"`
	FcSocketPath     string `long:"socket-path" short:"s" description:"path to use for firecracker socket"`
	TapMacAddr       string `long:"tap-mac-addr" description:"tap macaddress"`
	TapDev           string `long:"tap-dev" description:"tap device"`
	FcCPUCount       int64  `long:"ncpus" short:"c" description:"Number of CPUs"`
	FcMemSz          int64  `long:"memory" short:"m" description:"VM memory, in MiB"`
	FcIP             string `long:"fc-ip" description:"IP address of the VM"`

	IfName string `long:"if-name" description:"if name to match your main ethernet adapter,the one that accesses the Internet - check 'ip addr' or 'ifconfig' if you don't know which one to use"`
}

func (opts *options) getConfig() firecracker.Config {
	return firecracker.Config{
		VMID:            opts.Id,
		SocketPath:      opts.FcSocketPath,
		KernelImagePath: opts.FcKernelImage,
		KernelArgs:      opts.FckernelBootArgs,
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   &opts.FcRootDrivePath,
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
					HostDevName: opts.TapDev,
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

func installSignalHandlers(ctx context.Context, m *firecracker.Machine) {
	// not sure if this is actually really helping with anything
	go func() {
		// Clear some default handlers installed by the firecracker SDK:
		signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

		for {
			switch s := <-c; {
			case s == syscall.SIGTERM || s == os.Interrupt:
				log.Printf("Caught SIGINT, requesting clean shutdown")
				m.Shutdown(ctx)
			case s == syscall.SIGQUIT:
				log.Printf("Caught SIGTERM, forcing shutdown")
				m.StopVMM()
			}
		}
	}()
}
