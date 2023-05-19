package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
)

func (o *options) createVMM(ctx context.Context) (*Firecracker, error) {
	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	fcCfg := o.getConfig()

	cmd := firecracker.VMCommandBuilder{}.
		WithBin(o.FcBinary).
		WithSocketPath(fcCfg.SocketPath).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		Build(ctx)

	machineOpts := []firecracker.Opt{
		firecracker.WithProcessRunner(cmd),
	}

	// remove old socket path if it exists
	if _, err := RunNoneSudo(fmt.Sprintf("rm -f %s", o.ApiSocket)); err != nil {
		return nil, fmt.Errorf("failed to delete old socket path: %s", err)
	}

	// delete tap device if it exists
	if res, err := RunSudo(fmt.Sprintf("ip link del %s", o.Tap)); res != 1 && err != nil {
		return nil, fmt.Errorf("failed during deleting tap device: %v", err)
	}

	// create tap device
	if _, err := RunSudo(fmt.Sprintf("ip tuntap add dev %s mode tap", o.Tap)); err != nil {
		return nil, fmt.Errorf("failed creating ip link for tap: %s", err)
	}

	// set tap device mac address
	if _, err := RunSudo(fmt.Sprintf("ip addr add %s/24 dev %s", o.FcIP, o.Tap)); err != nil {
		return nil, fmt.Errorf("failed to add ip address on tap device: %v", err)
	}

	// set tap device up by activating it
	if _, err := RunSudo(fmt.Sprintf("ip link set %s up", o.Tap)); err != nil {
		return nil, fmt.Errorf("failed to set tap device up: %v", err)
	}

	// show tap connection created
	if _, err := RunNoneSudo(fmt.Sprintf("ip addr show dev %s", o.Tap)); err != nil {
		return nil, fmt.Errorf("failed to show tap connection created: %v", err)
	}

	//enable ip forwarding
	if _, err := RunSudo(" sh -c 'echo 1 > /proc/sys/net/ipv4/ip_forward'"); err != nil {
		return nil, fmt.Errorf("failed to enable ip forwarding: %v", err)
	}

	// add iptables rule to forward packets from tap to eth0
	if _, err := RunSudo(fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", o.IfName)); err != nil {
		return nil, fmt.Errorf("failed to add iptables rule to forward packets from tap to eth0: %v", err)
	}

	// add iptables rule to establish connection between tap and eth0 (forward packets from eth0 to tap)
	if _, err := RunSudo("iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT"); err != nil {
		return nil, fmt.Errorf("failed to add iptables rule to establish connection between tap and eth0: %v", err)
	}

	// add iptables rule to forward packets from eth0 to tap
	if _, err := RunSudo(fmt.Sprintf("iptables -A FORWARD -i %s -o %s -j ACCEPT", o.IfName, o.Tap)); err != nil {
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

	return &Firecracker{
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
	for _, run := range runVms {
		run.machine.StopVMM()
	}
}
