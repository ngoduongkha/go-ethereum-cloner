package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ngoduongkha/go-ethereum-cloner/database"
	"github.com/ngoduongkha/go-ethereum-cloner/node"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Launches the Ethereum node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrapIp, _ := cmd.Flags().GetString(flagBootstrapIp)
			bootstrapPort, _ := cmd.Flags().GetUint64(flagBootstrapPort)
			bootstrapAcc, _ := cmd.Flags().GetString(flagBootstrapAcc)

			fmt.Println("Launching Ethereum node and its HTTP API...")

			bootstrap := node.NewPeerNode(
				bootstrapIp,
				bootstrapPort,
				true,
				database.NewAccount(bootstrapAcc),
				false,
			)

			n := node.New(getDataDirFromCmd(cmd), ip, port, database.NewAccount(miner), bootstrap, node.DefaultMiningDifficulty)
			err := n.Run(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)
	addNodeHttpInfoFlags(runCmd)
	addMinerFlag(runCmd)
	addBootstrapInfoFlags(runCmd)

	return runCmd
}
