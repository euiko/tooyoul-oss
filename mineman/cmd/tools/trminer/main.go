package main

import (
	"context"
	"errors"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/miner"
	"github.com/euiko/tooyoul/mineman/pkg/miner/teamredminer"
	"github.com/spf13/cobra"
)

var (
	url          string
	user         string
	password     string
	algorithm    string
	path         string
	deviceById   int
	deviceByName string
	deviceByBus  string
	debug        bool
)

func main() {
	cmd := new(cobra.Command)
	cmd.RunE = run
	flags := cmd.Flags()
	flags.StringVar(&url, "url", "", "set the mining url")
	flags.StringVar(&user, "user", "", "user/wallet for mining")
	flags.StringVar(&user, "password", "x", "password for mining if exists")
	flags.StringVar(&algorithm, "algorithm", "", "algorithm to be used for mining")
	flags.StringVar(&path, "path", "", "custom path for the mining executeable")
	flags.IntVar(&deviceById, "device-by-id", -1, "select specific device by its index")
	flags.StringVar(&deviceByName, "device-by-name", "", "select specific device by its name")
	flags.StringVar(&deviceByBus, "device-by-bus", "", "select specific device by its bus")
	flags.BoolVar(&debug, "debug", false, "enable debug mode")

	if err := cmd.Execute(); err != nil {
		log.Error("failed to run command err : %s", log.WithValues(err))
	}
}

func run(cmd *cobra.Command, args []string) error {

	if url == "" {
		return errors.New("url must be specified")
	} else if user == "" {
		return errors.New("users must be specified")
	} else if algorithm == "" {
		return errors.New("algorithm must be specified")
	}

	// set debug level when enabled
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx := context.Background()
	config := config.NewViper("trminer", config.ViperStandalone())
	config.Set("url", url)
	config.Set("user", user)
	config.Set("password", password)

	executor := miner.NewPathExecutor(path)
	trminer := teamredminer.New()
	trminer.Init(ctx, config)

	options := []miner.OptionConfigurable{
		miner.WithAlgorithm(miner.Algorithm(algorithm)),
		miner.WithExecutor(executor),
	}

	if deviceById >= 0 {
		options = append(options, miner.WithDeviceByIndex(deviceById))
	} else if deviceByBus != "" {
		options = append(options, miner.WithDeviceByBus(deviceByBus))
	} else if deviceByName != "" {
		options = append(options, miner.WithDeviceByName(deviceByName))
	}

	trminer.Configure(options...)

	if err := trminer.Start(ctx); err != nil {
		return err
	}

	time.Sleep(time.Second * 20)
	return trminer.Stop()
}
