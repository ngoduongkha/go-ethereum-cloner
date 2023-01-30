package main

import (
	"fmt"
	"github.com/ngoduongkha/go-ethereum-cloner/node"
	"os"

	"github.com/ngoduongkha/go-ethereum-cloner/fs"
	"github.com/spf13/cobra"
)

const (
	flagKeystoreFile  = "keystore"
	flagDataDir       = "datadir"
	flagMiner         = "miner"
	flagIP            = "ip"
	flagPort          = "port"
	flagBootstrapAcc  = "bootstrap-account"
	flagBootstrapIp   = "bootstrap-ip"
	flagBootstrapPort = "bootstrap-port"
)

func main() {
	gethCmd := &cobra.Command{
		Use:   "geth",
		Short: "Go Ethereum CLI",
	}

	gethCmd.AddCommand(walletCmd())
	gethCmd.AddCommand(runCmd())

	err := gethCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to your node's data dir where the DB will be/is stored")
	_ = cmd.MarkFlagRequired(flagDataDir)
}

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagKeystoreFile, "", "Absolute path to the encrypted keystore file")
	_ = cmd.MarkFlagRequired(flagKeystoreFile)
}

func addMinerFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagMiner, "", "your node's miner account to receive the block rewards")
	_ = cmd.MarkFlagRequired(flagMiner)
}

func addNodeHttpInfoFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagIP, node.DefaultIP, "your node's public IP to communication with other peers")
	cmd.Flags().Uint64(flagPort, 0, "your node's public HTTP port for communication with other peers (configurable if SSL is disabled)")
	_ = cmd.MarkFlagRequired(flagPort)
}

func addBootstrapInfoFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagBootstrapIp, node.DefaultIP, "default bootstrap server to interconnect peers")
	cmd.Flags().Uint64(flagBootstrapPort, node.BootstrapPort, "default bootstrap server port to interconnect peers")
	cmd.Flags().String(flagBootstrapAcc, node.BootstrapAccount, "default bootstrap Genesis account with 1M ETH")
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)

	return fs.ExpandPath(dataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}
