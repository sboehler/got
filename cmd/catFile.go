// Package cmd is a command.
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/sboehler/got/pkg/repository"
	"github.com/sboehler/got/pkg/repository/object"
	"github.com/spf13/cobra"
)

// catFileCmd represents the catFile command
var catFileCmd = &cobra.Command{
	Use:   "cat-file TYPE OBJECT",
	Short: "Provide content of repository objects",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r, err := repository.Find(wd)
		if err != nil {
			return err
		}
		o, err := object.Load(r, object.Find(r, args[1], args[0], false))
		if err != nil {
			return err
		}
		if args[0] != o.Type() {
			return fmt.Errorf("bad object: %s", args[1])
		}
		_, err = io.Copy(cmd.OutOrStdout(), bytes.NewReader(o.Serialize()))
		return err
	},
	Args: cobra.ExactArgs(2),
}

func init() {
	rootCmd.AddCommand(catFileCmd)
}
