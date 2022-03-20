// Package cmd is a command.
package cmd

import (
	"bytes"
	"io"
	"os"

	"github.com/sboehler/got/pkg/repository"
	"github.com/spf13/cobra"
)

// catFileCmd represents the catFile command
var catFileCmd = &cobra.Command{
	Use:   "cat-file TYPE OBJECT",
	Short: "Provide content of repository objects",
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r, err := repository.Find(wd)
		if err != nil {
			return err
		}
		o, err := r.LoadObject(r.Find(args[1], args[0], false), args[0])
		if err != nil {
			return err
		}
		_, err = io.Copy(cmd.OutOrStdout(), bytes.NewReader(o.Serialize()))
		return err
	},
	Args: cobra.ExactArgs(2),
}

func init() {
	rootCmd.AddCommand(catFileCmd)
}
