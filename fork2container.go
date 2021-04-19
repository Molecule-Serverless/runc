// +build linux

package main

/*
#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/socket.h>
#include <sys/un.h>

int
sendfds(int s, int *fds, int fdcount); // defined in fork.go

// Send multiple FDs to the unix socket
int
sendMultipleFDs(char *sockPath, int chrootFD, int utsNamespaceFD, int pidNamespaceFD, int ipcNamespaceFD, int mntNamespaceFD) {
	// Connect to server via socket.
	int s, len, ret;
	struct sockaddr_un remote;

	if ((s = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
		return -1;
	}

	remote.sun_family = AF_UNIX;
	strcpy(remote.sun_path, sockPath);
	len = strlen(remote.sun_path) + sizeof(remote.sun_family);
	if (connect(s, (struct sockaddr *)&remote, len) == -1) {
		return -1;
	}

	int fds[5];
	fds[0] = chrootFD;
	fds[1] = utsNamespaceFD;
	fds[2] = pidNamespaceFD;
	fds[3] = ipcNamespaceFD;
	fds[4] = mntNamespaceFD;

	if (sendfds(s, fds, 5) == -1) {
		return -1;
	}

	char pid_arr[20];
	if (read(s, pid_arr, 20) < 0) {
		return -1;
	}

	int pid = atoi(pid_arr);

	if(close(s) == -1) {
		return -1;
	}

	return pid;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/runc/libcontainer/utils"
	"github.com/urfave/cli"
)

var fork2ContainerCommand = cli.Command{
	Name:        "fork2container",
	Usage:       "fork a process and land it in a container",
	ArgsUsage:   `TODO`,
	Description: `TODO`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "zygote",
			Value: "",
			Usage: `the container ID of the zygote container`,
		},
		cli.StringFlag{
			Name:  "target",
			Value: "",
			Usage: `the container ID of the target container to land the new process`,
		},
		cli.StringFlag{
			Name:  "fork-socket",
			Value: "rootfs/fork.sock",
			Usage: `the relative path to the fork socket in the zygote container according to the bundle path`,
		},
	},
	Action: func(context *cli.Context) error {
		endpointContainerID := context.String("target")
		if endpointContainerID == "" {
			return errors.New("template container not specified")
		}
		templateContainerID := context.String("zygote")
		if templateContainerID == "" {
			return errors.New("endpoint container not specified")
		}
		forkSocketPath := context.String("fork-socket")
		if forkSocketPath == "" {
			return errors.New("fork socket not specified")
		}
		err := doFork(context, templateContainerID, endpointContainerID, forkSocketPath)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return err
		}

		return nil
	},
}

func invokeMultipleFDs(socketPath string, rootDir *os.File, utsNamespaceFd *os.File, pidNamespaceFd *os.File, ipcNamespaceFd *os.File, mntNamespaceFd *os.File) (int, error) {
	cSock := C.CString(socketPath)
	defer C.free(unsafe.Pointer(cSock))
	pid, err := C.sendMultipleFDs(cSock, C.int(rootDir.Fd()), C.int(utsNamespaceFd.Fd()), C.int(pidNamespaceFd.Fd()), C.int(ipcNamespaceFd.Fd()), C.int(mntNamespaceFd.Fd()))
	if err != nil {
		return -1, err
	}
	return int(pid), nil
}

func doFork(context *cli.Context, zygoteContainerID string, targetContainerID string, forkSocketPath string) error {
	utils.UtilsPrintfLiu("start fork", "", "")
	// utils.UtilsPrintfLiu("before find target container id", "", "")

	targetContainer, err := getContainerByID(context, targetContainerID)
	// utils.UtilsPrintfLiu("find target container id", "", "")
	if err != nil {
		return err
	}
	zygoteContainer, err := getContainerByID(context, zygoteContainerID)

	if err != nil {
		return err
	}
	targetCgroupManager := targetContainer.GetCgroupsManager()
	if targetCgroupManager == nil {
		return errors.New("cgroups manager is nil")
	}
	targetContainerState, err := targetContainer.State()
	if err != nil {
		return err
	}
	zygoteContainerState, err := zygoteContainer.State()
	if err != nil {
		return err
	}
	if targetContainerState == nil {
		return errors.New("container state is nil")
	}
	// fmt.Println(targetContainerState.InitProcessPid)
	// utils.UtilsPrintfLiu("find containers by ids", "", "")

	// Open required namespace fds
	utsNamespace := "/proc/" + fmt.Sprint(targetContainerState.InitProcessPid) + "/ns/uts"
	pidNamespace := "/proc/" + fmt.Sprint(targetContainerState.InitProcessPid) + "/ns/pid"
	ipcNamespace := "/proc/" + fmt.Sprint(targetContainerState.InitProcessPid) + "/ns/ipc"
	mntNamespace := "/proc/" + fmt.Sprint(targetContainerState.InitProcessPid) + "/ns/mnt"
	utsNamespaceFd, err := os.Open(utsNamespace)
	if err != nil {
		return err
	}
	defer utsNamespaceFd.Close()
	pidNamespaceFd, err := os.Open(pidNamespace)
	if err != nil {
		return err
	}
	defer pidNamespaceFd.Close()
	ipcNamespaceFd, err := os.Open(ipcNamespace)
	if err != nil {
		return err
	}
	defer ipcNamespaceFd.Close()
	mntNamespaceFd, err := os.Open(mntNamespace)
	if err != nil {
		return err
	}
	defer mntNamespaceFd.Close()
	targetContainerBundle, _ := utils.Annotations(targetContainerState.Config.Labels)
	targetContainerRootfs, err := securejoin.SecureJoin(targetContainerBundle, "rootfs")
	if err != nil {
		return err
	}
	// fmt.Println(targetContainerRootfs)
	targetContainerRootfsFd, err := os.Open(targetContainerRootfs)
	if err != nil {
		return err
	}
	defer targetContainerRootfsFd.Close()

	// Find the path to the zygote container fork socket
	zygoteContainerBundle, _ := utils.Annotations(zygoteContainerState.Config.Labels)
	zygoteContainerForkSocketPath, err := securejoin.SecureJoin(zygoteContainerBundle, forkSocketPath)
	if err != nil {
		return err
	}
	// Send the fds to the socket
	pid, err := invokeMultipleFDs(zygoteContainerForkSocketPath, targetContainerRootfsFd, utsNamespaceFd, pidNamespaceFd, ipcNamespaceFd, mntNamespaceFd)
	if err != nil {
		return err
	}

	err = (*targetCgroupManager).Apply(pid)
	if err != nil {
		return err
	}
	utils.UtilsPrintfLiu("Apply cgroup to endpoint container", "", "")

	fmt.Println()
	utils.UtilsPrintfLiu("fork complete", "", "")
	return nil
}
