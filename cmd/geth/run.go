package main

import (
	"fmt"
	"github.com/ngoduongkha/go-ethereum-cloner/node"
	"github.com/spf13/cobra"
	"os"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launches the TBB node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {

			fmt.Println("Launching TBB node and its HTTP API...")

			err := node.Run()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	return runCmd
}
