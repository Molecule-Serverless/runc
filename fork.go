// +build linux

package main

import (
	"fmt"
	"net"
	"strconv"

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
		pid, err := invoke(socketPath)
		if err != nil {
			return err
		}
		fmt.Println(pid)
		fmt.Println("begin applying cgroups")
		err = (*cgroupsManager).Apply(pid)
		if err != nil {
			return err
		}
		config := (*newLinuxContainer).Config()
		err = (*newLinuxContainer).Set(config)
		if err != nil {
			return err
		}
		return nil
	},
}

func invoke(socketPath string) (int, error) {
	var pid int
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return -1, err
	}
	defer c.Close()

	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf[:])
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return -1, err
		}
		pid, err = strconv.Atoi(string(buf[0:n]))
		if err != nil {
			return -1, err
		}
	}
	return pid, nil
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
