// +build linux

package main

// #cgo CFLAGS: -ImoleculeOS-userlib/include -ImoleculeOS-userlib/local-ipc
// #cgo LDFLAGS: -L${SRCDIR}/moleculeOS-userlib -lmoleculeos
// void start_server();
import "C"

import (
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
		C.start_server()
		return nil
	},
}
