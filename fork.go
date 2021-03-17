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

char errmsg[1024];

int
sendfds(int s, int *fds, int fdcount) {
	char buf[1];
	struct iovec iov;
	struct msghdr header;
	struct cmsghdr *cmsg;
	int n;
	char cms[CMSG_SPACE(sizeof(int) * fdcount)];

	buf[0] = 0;
	iov.iov_base = buf;
	iov.iov_len = 1;

	memset(&header, 0, sizeof header);
	header.msg_iov = &iov;
	header.msg_iovlen = 1;
	header.msg_control = (caddr_t)cms;
	header.msg_controllen = CMSG_LEN(sizeof(int) * fdcount);

	cmsg = CMSG_FIRSTHDR(&header);
	cmsg->cmsg_len = CMSG_LEN(sizeof(int) * fdcount);
	cmsg->cmsg_level = SOL_SOCKET;
	cmsg->cmsg_type = SCM_RIGHTS;
	memmove(CMSG_DATA(cmsg), fds, sizeof(int) * fdcount);

	if((n = sendmsg(s, &header, 0)) != iov.iov_len) {
		return -1;
	}

	return 0;
}

int
sendRootFD(char *sockPath, int chrootFD) {
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

	int fds[1];
	fds[0] = chrootFD;
	if (sendfds(s, fds, 1) == -1) {
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
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/utils"
	"github.com/urfave/cli"
)

var forkCommand = cli.Command{
	Name:  "fork",
	Usage: "fork a container",
	ArgsUsage: `<container-id>

	Where "<container-id>" is your name for the instance of the container that you
	are starting. The name you provide for the container instance must be unique on
	your host.`,
	Description: `test`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bundle, b",
			Value: "",
			Usage: `path to the root of the bundle directory, defaults to the current directory`,
		},
	},
	Action: func(context *cli.Context) error {
		container, err := getContainer(context)
		if err != nil {
			return err
		}

		newContainerRootfs := context.String("bundle")
		newContainerRootfs = filepath.Join(newContainerRootfs, "rootfs")
		newContainerRootfsFd, err := os.Open(newContainerRootfs)
		if err != nil {
			return err
		}
		defer newContainerRootfsFd.Close()

		state, err := container.State()
		bundle, _ := utils.Annotations(state.Config.Labels)
		socketName := context.Args()[1]
		newContainerID := context.Args()[2]
		newLinuxContainer, err := loadDefaultContainer(context, newContainerID)
		if err != nil {
			return err
		}
		cgroupsManager := (*newLinuxContainer).GetCgroupsManager()
		fmt.Println(bundle)
		socketPath, err := securejoin.SecureJoin(bundle, socketName)
		if err != nil {
			return err
		}
		pid, err := invoke(socketPath, newContainerRootfsFd)
		if err != nil {
			return err
		}
		// fmt.Println(pid)
		// fmt.Println("begin applying cgroups")

		err = (*cgroupsManager).Apply(pid)
		if err != nil {
			return err
		}

		err = (*newLinuxContainer).InitializeFakeContainer(pid)
		if err != nil {
			return err
		}

		config := (*newLinuxContainer).Config()
		err = (*newLinuxContainer).Set(config)
		if err != nil {
			return err
		}

		/*newContainerState, err := (*newLinuxContainer).State()
		if err != nil {
			return err
		}
		if newContainerStateString, err := json.Marshal(newContainerState); err == nil {
			fmt.Println(string(newContainerStateString))
		} else {
			return err
		}*/

		return nil
	},
}

func invoke(socketPath string, rootDir *os.File) (int, error) {
	cSock := C.CString(socketPath)
	defer C.free(unsafe.Pointer(cSock))
	pid, err := C.sendRootFD(cSock, C.int(rootDir.Fd()))
	if err != nil {
		return -1, err
	}
	return int(pid), nil
}

func loadDefaultContainer(context *cli.Context, id string) (*libcontainer.Container, error) {
	spec, err := setupSpec(context)
	if err != nil {
		return nil, err
	}
	fmt.Printf("create a new container %s\n", id)
	container, err := createContainer(context, id, spec)
	if err != nil {
		return nil, err
	}
	return &container, nil
}
