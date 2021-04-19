// +build linux

package main

// #cgo CFLAGS: -ImoleculeOS-userlib/include -ImoleculeOS-userlib/local-ipc
// #cgo LDFLAGS: -L${SRCDIR}/moleculeOS-userlib -lmoleculeos
// #include <molecule_runtime_command_protocol.h>
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
)

var runtimeCommand = cli.Command{
	Name:        "runtime",
	Usage:       "wait for a Molecule global-fifo to send commands",
	ArgsUsage:   `TODO`,
	Description: `TODO`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "zygote",
			Value: "",
			Usage: `the container ID of the zygote container`,
		},
		//TODO: need to include default value of arguments to execute `runc run`
	},
	Action: func(context *cli.Context) error {
		var fifo_self, global_fifo C.int
		C.init_server(&fifo_self, &global_fifo)

		for {
			args := C.GoString(C.receive_from_server(fifo_self, global_fifo))
			infos := strings.Split(args, ";")
			fmt.Printf("%q\n", infos)
			command, err := getCommand(infos)
			if err != nil {
				fmt.Printf("error: %s", err)
				os.Exit(-1)
			}
			fmt.Printf("command: %s\n", command)
			switch command {
			case "cfork":
				go doCforkCommand(context, infos[1:])
				// if err != nil {
				// 	fmt.Printf("error: %s", err)
				// 	os.Exit(-1)
				// }
			default:
				go doRunCommand(context, infos[1:])
				// if err != nil {
				// 	fmt.Printf("error: %s", err)
				// 	os.Exit(-1)
				// }
			}

		}
		return nil
	},
}

func getCommand(infos []string) (string, error) {
	command := strings.Split(infos[0], ":")
	if len(command) != 2 || command[0] != "command" {
		return "", errors.New("invalid message")
	}
	switch command[1] {
	case "cfork", "run":
		return command[1], nil
	default:
		return "", errors.New("invalid message")
	}
}

func doCforkCommand(context *cli.Context, rawArgs []string) error {
	//TODO: parse cfork args
	templateContainerID, endpointContainerID, err := parseCforkArgs(rawArgs)
	if err != nil {
		return err
	}
	doFork(context, templateContainerID, endpointContainerID, "rootfs/fork.sock")
	//TODO: run cfork
	return nil
}

func parseCforkArgs(rawArgs []string) (templateContainerID string, endpointContainerID string, err error) {
	templateContainerID = ""
	endpointContainerID = ""
	err = nil
	if len(rawArgs) != 2 {
		err = errors.New("no enough args")
		return
	}
	rawTmpArg := strings.Split(rawArgs[0], ":")
	if len(rawTmpArg) != 2 || rawTmpArg[0] != "template" {
		err = errors.New("wrong template container id")
		return
	}
	templateContainerID = rawTmpArg[1]

	rawEpArg := strings.Split(rawArgs[1], ":")
	if len(rawEpArg) != 2 || rawEpArg[0] != "endpoint" {
		err = errors.New("wrong endpoint container id")
		return
	}
	endpointContainerID = rawEpArg[1]

	return
}

func parseRunArgs(rawArgs []string) (containerID string, bundle string, err error) {
	containerID = ""
	bundle = ""
	err = nil
	if len(rawArgs) != 2 {
		err = errors.New("no enough args")
		return
	}
	rawCidArg := strings.Split(rawArgs[0], ":")
	if len(rawCidArg) != 2 || rawCidArg[0] != "containerID" {
		err = errors.New("wrong container id")
		return
	}
	containerID = rawCidArg[1]
	rawBundleArg := strings.Split(rawArgs[1], ":")
	if len(rawBundleArg) != 2 || rawBundleArg[0] != "bundle" {
		err = errors.New("wrong bundle")
		return
	}
	bundle = rawBundleArg[1]

	return
}

func doRunCommand(context *cli.Context, rawArgs []string) error {
	containerID, bundle, err := parseRunArgs(rawArgs)
	if err != nil {
		return err
	}

	if bundle != "" {
		if err := os.Chdir(bundle); err != nil {
			return err
		}
	}
	spec, err := loadSpec(specConfig)
	if err != nil {
		return err
	}
	_, err = startContainerWithID(context, containerID, spec, CT_ACT_RUN, nil)
	// status, err := startContainerWithID(context, containerID, spec, CT_ACT_RUN, nil)
	// if err == nil {
	// 	os.Exit(status)
	// }
	return err
}
