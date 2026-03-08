package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var gendocsCmd = &cobra.Command{
	Use:    "gendocs",
	Short:  "Generate man pages",
	Long:   `Generate man pages for the linear CLI into a specified directory.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		header := &doc.GenManHeader{
			Title:   "LINEAR",
			Section: "1",
			Source:  "linear-cli",
		}
		if err := doc.GenManTree(rootCmd, header, dir); err != nil {
			return err
		}
		fmt.Printf("Man pages generated in %s\n", dir)
		return nil
	},
}

func init() {
	gendocsCmd.Flags().String("dir", "./man", "Output directory for man pages")
	rootCmd.AddCommand(gendocsCmd)
}
