// rootfs file is used to generate root filesystem for the VM
// using init binary process and supplied tar file form docker supplied by user.
package main

import (
	"fmt"
	"os"
)

// GenerateRFs generates root filesystem for the VM according to the below steps:
// 1. create a directory for the rootfs
// 2. copy the init binary to the rootfs
// 3. copy the init base tar file to the rootfs
// 4. extract the init base tar file
// 5. copy the docker supplied tar file to the rootfs
// 6. extract the docker supplied tar file
// 7. delete the init base tar file
// 8. delete the docker supplied tar file
// 9. return the rootfs path or name
func (o *options) GenerateRFs(name string) (string, error) {

	fsName := fmt.Sprintf("%d-%s.ext4", o.VmIndex, name)

	// for creating the rootfs directory with 526MB size
	if _, err := RunNoneSudo(fmt.Sprintf("fallocate -l 526MB %s", fsName)); err != nil {
		return "", fmt.Errorf("failed to create rootfs file: %v", err)
	}

	//for making the rootfs file as ext4 file system
	if _, err := RunNoneSudo(fmt.Sprintf("mkfs.ext4 %s", fsName)); err != nil {
		return "", fmt.Errorf("failed to create ext4 file system: %v", err)
	}

	//creating a temporary directory for mounting the rootfs file
	tmpDir, err := os.MkdirTemp("", fsName)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// unmout the created tmp dir from rootfs file
	defer RunSudo(fmt.Sprintf("umount %s", tmpDir))

	//for mounting the created rootfs file to tmp directory
	if _, err := RunSudo(fmt.Sprintf("mount -o loop %s %s", fsName, tmpDir)); err != nil {
		return "", fmt.Errorf("failed to mount rootfs file: %v", err)
	}

	imageTar := fmt.Sprintf("%d-%s.tar", o.VmIndex, name)
	imageName := fmt.Sprintf("%d-%s", o.VmIndex, name)

	// for exporting the docker tar file from supplied docker image
	if _, err := RunNoneSudo(fmt.Sprintf("docker create --name %s %s", imageName, o.ProvidedImage)); err != nil {
		return "", fmt.Errorf("failed to export docker tar file: %v", err)
	}
	defer RunNoneSudo(fmt.Sprintf("docker rm -f %s", imageName))

	// for exporting the docker tar file from supplied docker image
	if _, err := RunNoneSudo(fmt.Sprintf("docker export %s -o %s", imageName, imageTar)); err != nil {
		return "", fmt.Errorf("failed to export docker tar file: %v", err)
	}

	// for extracting the docker supplied tar file to the rootfs directory
	if _, err := RunSudo(fmt.Sprintf("tar -xvf %s -C %s", imageTar, tmpDir)); err != nil {
		return "", fmt.Errorf("failed to extract docker supplied tar file: %v", err)
	}

	// include our init process into ext4 file system exported from docker
	if _, err := RunSudo(fmt.Sprintf("cp -r init %s", tmpDir)); err != nil {
		return "", fmt.Errorf("failed to cp init to tmp dir: %v", err)
	}

	//remove those created ext and tar files
	if _, err := RunNoneSudo(fmt.Sprintf("rm -f %s", imageTar)); err != nil {
		return "", fmt.Errorf("failed to remove ext and tar files: %v", err)
	}

	return fsName, nil
}
