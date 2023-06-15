package main

import "fmt"

func (o *options) SetNetwork() error {

	// delete tap device if it exists
	if res, err := RunSudo(fmt.Sprintf("ip link del %s 2> /dev/null || true", o.Tap)); res != 1 && err != nil {
		return fmt.Errorf("failed during deleting tap device: %v", err)
	}

	// create tap device
	if _, err := RunSudo(fmt.Sprintf("ip tuntap add dev %s mode tap", o.Tap)); err != nil {
		return fmt.Errorf("failed creating ip link for tap: %s", err)
	}

	// set tap device mac address
	if _, err := RunSudo(fmt.Sprintf("ip addr add %s/24 dev %s", o.FcIP, o.Tap)); err != nil {
		return fmt.Errorf("failed to add ip address on tap device: %v", err)
	}

	// set tap device up by activating it
	if _, err := RunSudo(fmt.Sprintf("ip link set %s up", o.Tap)); err != nil {
		return fmt.Errorf("failed to set tap device up: %v", err)
	}

	//set master bridge for tap device
	// if _, err := RunSudo(fmt.Sprintf("ip link set %s master docker0", o.Tap)); err != nil {
	// 	return fmt.Errorf("failed to set master bridge for tap device: %v", err)
	// }

	if _, err := RunSudo(fmt.Sprintf("sysctl -w net.ipv4.conf.%s.proxy_arp=1", o.Tap)); err != nil {
		return fmt.Errorf("failed doing first sysctl: %v", err)
	}

	if _, err := RunSudo(fmt.Sprintf("sysctl -w net.ipv6.conf.%s.disable_ipv6=1", o.Tap)); err != nil {
		return fmt.Errorf("failed doing second sysctl: %v", err)
	}

	//enable ip forwarding
	if _, err := RunSudo(" sh -c 'echo 1 > /proc/sys/net/ipv4/ip_forward'"); err != nil {
		return fmt.Errorf("failed to enable ip forwarding: %v", err)
	}

	// add iptables rule to forward packets from tap to eth0
	if _, err := RunSudo(fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", o.BackBone)); err != nil {
		return fmt.Errorf("failed to add iptables rule to forward packets from tap to eth0: %v", err)
	}

	// add iptables rule to establish connection between tap and eth0 (forward packets from eth0 to tap)
	if _, err := RunSudo("iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT"); err != nil {
		return fmt.Errorf("failed to add iptables rule to establish connection between tap and eth0: %v", err)
	}

	// add iptables rule to forward packets from eth0 to tap
	if _, err := RunSudo(fmt.Sprintf("iptables -A FORWARD -i %s -o %s -j ACCEPT", o.Tap, o.BackBone)); err != nil {
		return fmt.Errorf("failed to add iptables rule to forward packets from eth0 to tap: %v", err)
	}

	return nil

}
