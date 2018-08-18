package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/diegobernardes/flare/internal/application/api"
)

var defaultConfigPath = "./flare.toml"

func main() {
	var rootCmd = &cobra.Command{Use: "flare"}
	rootCmd.AddCommand(commandStart(), commandSetup(), commandVersion())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(errors.Wrap(err, "error during Flare command initialization"))
		os.Exit(1)
	}
}

func commandStart() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the service",
		Long:  "Command used to start the service.",
		Run: func(cmd *cobra.Command, args []string) {
			client := newClient(configPath)

			if err := client.Start(); err != nil {
				fmt.Println(errors.Wrap(err, "error during client start"))
				os.Exit(1)
			}

			chanExit := make(chan os.Signal, 1)
			signal.Notify(chanExit, os.Interrupt)
			<-chanExit

			if err := client.Stop(); err != nil {
				fmt.Println(errors.Wrap(err, "error during client stop"))
				os.Exit(1)
			}
		},
	}
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "")

	return cmd
}

func commandSetup() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup the required resources",
		Long:  "Based at the configuration, it run the setup on all required resources.",
		Run: func(cmd *cobra.Command, args []string) {
			client := newClient(configPath)

			ctx, ctxCancel := context.WithCancel(context.Background())
			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt)
				<-c
				ctxCancel()
			}()

			if err := client.Setup(ctx); err != nil {
				fmt.Println(errors.Wrap(err, "error during client setup"))
				os.Exit(1)
			}
		},
	}
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "")

	return cmd
}

func commandVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version",
		Long:  "Show information about the Go runtime and Flare version.",
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

			if api.Version != "" {
				fmt.Fprintln(w, fmt.Sprintf("Version:\t%s", api.Version))
			}

			if api.Commit != "" {
				fmt.Fprintln(w, fmt.Sprintf("Commit:\t%s", api.Commit))
			}

			if api.BuildTime != "" {
				fmt.Fprintln(w, fmt.Sprintf("Build Time:\t%s", api.BuildTime))
			}

			fmt.Fprintln(w, fmt.Sprintf("Go Version:\t%s", api.GoVersion))

			if err := w.Flush(); err != nil {
				fmt.Println(errors.Wrap(err, "error during version output write"))
				os.Exit(1)
			}
		},
	}
}

func readConfig(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if path != defaultConfigPath {
			return "", err
		}
		return "", nil
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func newClient(path string) api.Client {
	config, err := readConfig(path)
	if err != nil {
		fmt.Println(errors.Wrap(err, "could not load configuration file"))
		os.Exit(1)
	}

	client := api.Client{
		Config: config,
	}

	if err := client.Init(); err != nil {
		fmt.Println(errors.Wrap(err, "error during client initialization"))
		os.Exit(1)
	}
	return client
}
