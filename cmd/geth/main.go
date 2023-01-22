package main

import (
	"fmt"
	"os"

	"github.com/ngoduongkha/go-ethereum-cloner/fs"
	"github.com/spf13/cobra"
)

const (
	flagKeystoreFile  = "keystore"
	flagDataDir       = "datadir"
	flagMiner         = "miner"
	flagSSLEmail      = "ssl-email"
	flagDisableSSL    = "disable-ssl"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to your node's data dir where the DB will be/is stored")
	cmd.MarkFlagRequired(flagDataDir)
}

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagKeystoreFile, "", "Absolute path to the encrypted keystore file")
	cmd.MarkFlagRequired(flagKeystoreFile)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)

	return fs.ExpandPath(dataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}
