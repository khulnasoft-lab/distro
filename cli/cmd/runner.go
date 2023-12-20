package cmd

import (
	"github.com/khulnasoft-lab/distro/services/runners"
	"github.com/khulnasoft-lab/distro/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runnerCmd)
}

func runRunner() {
	util.ConfigInit(configPath)

	taskPool := runners.JobPool{}

	taskPool.Run()
}

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Run in runner mode",
	Run: func(cmd *cobra.Command, args []string) {
		runRunner()
	},
}
