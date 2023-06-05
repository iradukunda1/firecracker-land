package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"syscall"
)

const paths = "PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"

// main starts an init process that can prepare an environment and start a shell
// after the Kernel has started.
func main() {
	fmt.Printf("Drop init booting\nCopyright Quark-Group 2023, OpenFaaS Ltd\n")

	mount("none", "/proc", "proc", 0)
	mount("none", "/dev/pts", "devpts", 0)
	mount("none", "/dev/mqueue", "mqueue", 0)
	mount("none", "/dev/shm", "tmpfs", 0)
	mount("none", "/sys", "sysfs", 0)
	mount("none", "/sys/fs/cgroup", "cgroup", 0)

	setHostname("drop-vm")

	fmt.Printf("Drop starting /bin/sh\n")

	cmd := exec.Command("/bin/sh")

	cmd.Env = append(cmd.Env, paths)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("could not start /bin/sh, error: %s", err))
	}

	err = cmd.Wait()
	if err != nil {
		panic(fmt.Sprintf("could not wait for /bin/sh, error: %s", err))
	}
}

func setHostname(hostname string) {
	err := syscall.Sethostname([]byte(hostname))
	if err != nil {
		panic(fmt.Sprintf("cannot set hostname to %s, error: %s", hostname, err))
	}
}

func mount(source, target, filesystemtype string, flags uintptr) {

	if _, err := os.Stat(target); os.IsNotExist(err) {
		err := os.MkdirAll(target, 0755)
		if err != nil {
			panic(fmt.Sprintf("error creating target folder: %s %s", target, err))
		}
	}

	err := syscall.Mount(source, target, filesystemtype, flags, "")
	if err != nil {
		log.Printf("%s", fmt.Errorf("error mounting %s to %s, error: %s", source, target, err))
	}
}
