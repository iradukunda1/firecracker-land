package main

import (
	"io/ioutil"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func main1() {

	lg := log.New()

	tmpDir, err := ioutil.TempDir("", "container")
	if err != nil {
		lg.Errorln("Create a temporary directory to hold the container files %v", err)
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	lg.Infoln("Start a container from the Docker image")

	cmd := exec.Command("docker", "run", "--name=mycontainer", "0780633340/emma-rwanda", "sh", "-c", "apk update && apk add --no-cache nginx")
	cmd.Run()

	lg.Infoln("Copy the contents of the container to the temporary directory")
	cmd = exec.Command("docker", "cp", "mycontainer:/", tmpDir)
	cmd.Run()

	lg.Infoln("Create a disk image file")

	imgFile, err := os.Create("myimage.ext4")
	if err != nil {
		lg.Errorln("error during creating image %v", err)
		panic(err)
	}
	defer imgFile.Close()

	lg.Infoln("Set the size of the disk image file to 1GB")
	imgFile.Truncate(1024 * 1024 * 1024)

	cmd = exec.Command("mkfs.ext4", "-F", "myimage.ext4")
	cmd.Run()

	lg.Infoln("Mount the disk image file")
	// Mount the disk image file
	mountPoint, err := ioutil.TempDir("", "mnt")
	if err != nil {
		lg.Errorln("error during Mount the disk image file %v", err)
		panic(err)
	}
	defer os.RemoveAll(mountPoint)

	cmd = exec.Command("mount", "-o", "loop", "myimage.ext4", mountPoint)
	cmd.Run()

	lg.Infoln("Copy the contents of the temporary directory to the mounted disk image file")

	cmd = exec.Command("cp", "-r", tmpDir+"/.", mountPoint)
	cmd.Run()

	lg.Infoln("Install the boot loader to the disk image file")

	cmd = exec.Command("grub-install", "--root-directory="+mountPoint, "--no-floppy", "--modules=part_msdos", "/dev/loop0")
	cmd.Run()

	lg.Infoln("Unmount the disk image file")

	cmd = exec.Command("umount", mountPoint)
	cmd.Run()

	lg.Infoln("Bootable image created ")
}
