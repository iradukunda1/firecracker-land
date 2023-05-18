package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/iradukunda1/utils"
	log "github.com/sirupsen/logrus"
)

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

func cleanup() {
	for _, running := range runningVMs {
		running.machine.StopVMM()
	}
}
