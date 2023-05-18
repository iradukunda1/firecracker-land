package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func mains() {
	lg := log.New()

	// Create a temporary directory to hold the container files
	tmpDir, err := ioutil.TempDir("", "container")
	if err != nil {
		lg.Fatalf("Failed to create temporary directory: %v", err)
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// Start a container from the Docker image
	lg.Infoln("Starting a container from the Docker image")
	cmd := exec.Command("docker", "run", "--name=mycontainer", "0780633340/rust-http", "sh")
	if err := cmd.Run(); err != nil {
		exec.Command("docker", "rm", "mycontainer", "-f").Run()
		lg.Fatalf("Failed to start container: %v", err)
	}

	lg.Infoln("The temp file is", tmpDir)
	// Copy the contents of the container to the mounted disk image file
	lg.Infoln("Copying the contents of the container to the mounted disk image file")

	cmd = exec.Command("docker", "cp", "mycontainer:/", tmpDir)
	if err := cmd.Run(); err != nil {
		lg.Fatalf("Failed to copy files to disk image: %v", err)
	}

	// Remove unnecessary packages and files from the mounted disk image file
	lg.Infoln("Removing unnecessary packages and files from the mounted disk image file")
	cmd = exec.Command("docker", "run", "--rm", "-v", tmpDir+":/rootfs", "alpine", "sh", "-c", "apk del --purge apk-tools && rm -rf /etc/apk && rm -rf /var/cache/apk/*")
	if err := cmd.Run(); err != nil {
		exec.Command("docker", "rm", "mycontainer", "-f").Run()
		lg.Warnf("Failed to remove unnecessary packages and files: %v", err)
		panic(err)
	}

	// Determine the required size of the disk image file
	lg.Infoln("Determining the required size of the disk image file")
	var size int64
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		exec.Command("docker", "rm", "mycontainer", "-f").Run()
		lg.Fatalf("Failed to calculate required size of disk image: %v", err)
		panic(err)
	}

	// Create the disk image file with the required size
	lg.Infoln("Creating the disk image file with the required size")
	imgFile, err := os.Create("myimage.ext4")
	if err != nil {
		lg.Fatalf("Failed to create image file: %v", err)
		panic(err)
	}
	defer imgFile.Close()

	if err := imgFile.Truncate(1024 * 1024 * 1024); err != nil {
		lg.Errorln("error during creating image size %v", err)
		panic(err)
	}

	cmd = exec.Command("mkfs.ext4", "-F", "myimage.ext4")
	if err := cmd.Run(); err != nil {
		lg.Errorln("error during creating ext4 bootable %v", err)
		panic(err)
	}

	lg.Infoln("Mount the disk image file")

	// Mount the disk image file
	mountPoint, err := ioutil.TempDir("", "mnt")
	if err != nil {
		lg.Errorln("error during Mount the disk image file %v", err)
		panic(err)
	}
	defer os.RemoveAll(mountPoint)

	lg.Infoln("Mount file location", mountPoint)

	cmd = exec.Command("sudo", "mount", "-o", "loop", "myimage.ext4", mountPoint)
	if err := cmd.Run(); err != nil {
		lg.Errorln("error during mounting ext4 rootfs %v", err)
		panic(err)
	}

	// lg.Infoln("Copy the contents of the temporary directory to the mounted disk image file")

	// os.Exit(0)

	// cmd = exec.Command("sudo", "cp", "-r", tmpDir+"/.", mountPoint) //I have to refactor this cause it is using sudo previlage
	// if err := cmd.Run(); err != nil {
	// 	lg.Errorln("error during copying temp file to mount file %v", err)
	// 	panic(err)
	// }

	// lg.Infoln("Install the boot loader to the disk image file")

	// cmd = exec.Command("grub-install", "--root-directory="+mountPoint, "--no-floppy", "--modules=part_msdos", "/dev/loop0")
	// cmd.Run()

	lg.Infoln("Unmount the disk image file")

	cmd = exec.Command("sudo", "umount", mountPoint)
	if err := cmd.Run(); err != nil {
		lg.Errorln("error during unmounting ext4 file %v", err)
		panic(err)
	}

	lg.Infoln("Bootable image created ")
}

func resizeMountPoint(mountPoint string, size int64) error {
	cmd := exec.Command("dd", "if=/dev/zero", "of="+mountPoint+"/dummyfile", "bs="+strconv.FormatInt(size, 10), "count=1")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("rm", "-f", mountPoint+"/dummyfile")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
