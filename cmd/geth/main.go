package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	var gethCmd = &cobra.Command{
		Use:   "geth",
		Short: "The go-ethereum CLI",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	gethCmd.AddCommand(balancesCmd())
	gethCmd.AddCommand(runCmd())

	err := gethCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}
