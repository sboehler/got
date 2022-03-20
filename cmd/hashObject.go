/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/

// Package cmd implements commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/sboehler/got/pkg/object"
	"github.com/sboehler/got/pkg/repository"
	"github.com/spf13/cobra"
)

// hashObjectCmd represents the hashObject command
var (
	objectType string
	write      bool

	hashObjectCmd = &cobra.Command{
		Use:   "hash-object OBJECT",
		Short: "Provide content of repository objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var o repository.Object
			switch objectType {
			case "blob":
				o = object.NewBlob(f)
			default:
				return fmt.Errorf("invalid object type: %s", objectType)
			}
			of := &repository.ObjectFile{
				Data:       o.Serialize(),
				ObjectType: objectType,
			}
			var hash string
			if write {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				r, err := repository.Find(wd)
				if err != nil {
					return err
				}
				if hash, err = r.WriteObject(of); err != nil {
					return err
				}
			} else {
				hash = repository.Hash(of)
			}
			fmt.Println(hash)
			return nil
		},
		Args: cobra.ExactArgs(1),
	}
)

func init() {
	hashObjectCmd.Flags().StringVarP(&objectType, "type", "t", "blob", "specify tye type")
	hashObjectCmd.Flags().BoolVarP(&write, "write", "w", false, "write the file to the object database")
	rootCmd.AddCommand(hashObjectCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// hashObjectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// hashObjectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
