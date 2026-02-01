package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "ralph-cc [file]",
		Short: "ralph-cc is a C compiler frontend for testing compilation passes",
		Long: `ralph-cc is a C compiler frontend CLI optimized for testing
compilation passes rather than practical use. It follows the
CompCert design with the goal of equivalent output on each IR.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
			filename := args[0]
			fmt.Fprintf(os.Stderr, "ralph-cc: compiling %s\n", filename)
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
