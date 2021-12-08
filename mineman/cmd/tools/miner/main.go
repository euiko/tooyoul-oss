package main

import (
	"context"
	"errors"
	"os"

	"github.com/euiko/tooyoul/mineman/pkg/config"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/miner"
	"github.com/euiko/tooyoul/mineman/pkg/miner/teamredminer"
	"github.com/euiko/tooyoul/mineman/pkg/runner"
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
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		trminer, err := load(ctx, cmd, args)
		if err != nil {
			return err
		}

		runner.Run(ctx, runner.OperationFunc(func(_ context.Context) error {
			return trminer.Start(ctx)
		})).
			OnSignal(runner.SignalHandlerFunc(func(_ context.Context, sig os.Signal) {
				trminer.Stop()
			})).
			Wait(ctx)

		return nil
	}
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

func load(ctx context.Context, cmd *cobra.Command, args []string) (*teamredminer.Miner, error) {

	if url == "" {
		return nil, errors.New("url must be specified")
	} else if user == "" {
		return nil, errors.New("users must be specified")
	} else if algorithm == "" {
		return nil, errors.New("algorithm must be specified")
	}

	// set debug level when enabled
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	config := config.NewViper("trminer", config.ViperStandalone())
	executor := miner.NewPathExecutor(path)
	trminer := teamredminer.New()
	trminer.Init(ctx, config)

	options := []miner.OptionConfigurable{
		miner.WithPool(miner.Pool{
			Algorithm: miner.Kawpow,
			Url:       url,
			User:      user,
			Pass:      password,
		}),
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
	return trminer, nil
}
