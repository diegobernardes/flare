package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/diegobernardes/flare/services/flare"
)

func main() {
	var configPath string

	var cmdStart = &cobra.Command{
		Use:   "start",
		Short: "Start Flare service",
		Long: `This command is used to start the Flare service. The application gonna
look for a 'flare.toml' file at the same directory as the binary.`,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := readConfig(configPath)
			if err != nil && configPath != "./flare.toml" {
				fmt.Println(errors.Wrap(err, "could not load configuration file"))
				os.Exit(1)
			}

			client := flare.NewClient(flare.ClientConfig(config))
			if err := client.Start(); err != nil {
				os.Exit(1)
			}

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c

			if err := client.Stop(); err != nil {
				os.Exit(1)
			}
		},
	}
	cmdStart.PersistentFlags().StringVarP(&configPath, "config", "c", "./flare.toml", "")

	var rootCmd = &cobra.Command{Use: "flare"}
	rootCmd.AddCommand(cmdStart)
	rootCmd.Execute()
}

func readConfig(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
