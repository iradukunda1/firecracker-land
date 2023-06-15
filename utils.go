package main

import (
	"os"
)

// exposeBlockDeviceToJail will call mknod on the block device to ensure
// visibility of the device
func exposeBlockDeviceToJail(dst string, uid, gid int) error {

	//since it was already create there is no need to recreate it which can cause exists error
	// if err := syscall.Mknod(dst, syscall.S_IFBLK, rdev); err != nil {
	// 	return err
	// }

	if err := os.Chmod(dst, 0600); err != nil {
		return err
	}

	if err := os.Chown(dst, uid, gid); err != nil {
		return err
	}

	return nil
}
