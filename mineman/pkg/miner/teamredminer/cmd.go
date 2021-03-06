package teamredminer

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var commandBuilderRegistry []CommandBuilder

type (
	command interface {
		SendErr(err error)
	}
	commandStart struct {
		errChan chan error
	}
	commandStop struct {
		errChan chan error
	}

	CommandBuilder func(miner *Miner, args []string) ([]string, error)
)

func (cmd *commandStart) SendErr(err error) {
	cmd.errChan <- err
}
func (cmd *commandStop) SendErr(err error) {
	cmd.errChan <- err
}

func RegisterCommandBuilder(builder CommandBuilder) {
	commandBuilderRegistry = append(commandBuilderRegistry, builder)
}

func newCommandStart() *commandStart {
	return &commandStart{
		errChan: make(chan error),
	}
}

func newCommandStop() *commandStop {
	return &commandStop{
		errChan: make(chan error),
	}
}

func poolArgsBuilder(miner *Miner, args []string) ([]string, error) {
	pool := miner.settings.Pool

	if pool.Url == "" {
		return nil, errors.New("pool url are required for mining")
	}
	if pool.User == "" {
		return nil, errors.New("pool user are required for mining")
	}

	pass := pool.Pass
	if pass == "" {
		pass = "x"
	}

	args = append(args,
		"-u", pool.User,
		"-o", pool.Url,
		"-p", pass,
	)

	return args, nil
}

func algorithmArgsBuilder(miner *Miner, args []string) ([]string, error) {
	option := miner.settings
	algorithm := option.Pool.Algorithm

	supportedAlgorithms := miner.Algorithms()
	algoritmStrings := make([]string, len(supportedAlgorithms))
	for i, a := range supportedAlgorithms {
		algoritmStrings[i] = string(a)
	}

	sort.Strings(algoritmStrings)
	if i := sort.SearchStrings(algoritmStrings, string(algorithm)); i >= len(algoritmStrings) {
		return nil, fmt.Errorf("algorithm of %s is not supported by %s", algorithm, name)
	}

	args = append(args, "-a", string(algorithm))

	return args, nil
}

func deviceArgsBuilder(miner *Miner, args []string) ([]string, error) {
	var selectedGpus []string
	option := miner.settings

	// build devices arguments
	if option.Device != nil {
		devices, err := miner.Select(option.Device, &selectedGpus)
		if err != nil {
			return nil, err
		}

		for devices.Next() {
			var deviceId int
			if err := devices.Scan(&deviceId); err != nil {
				return nil, err
			}
			selectedGpus = append(selectedGpus, strconv.Itoa(deviceId))
		}
	}
	if len(selectedGpus) >= 0 {
		args = append(args, "-d", strings.Join(selectedGpus, ","))
	}

	return args, nil
}

func BuildCommandArgs(miner *Miner) ([]string, error) {
	var (
		args []string
		err  error
	)

	// build algorithm using registered command builder
	for _, b := range commandBuilderRegistry {
		args, err = b(miner, args)
		if err != nil {
			return nil, err
		}
	}

	return args, nil
}

func init() {
	RegisterCommandBuilder(poolArgsBuilder)
	RegisterCommandBuilder(deviceArgsBuilder)
	RegisterCommandBuilder(algorithmArgsBuilder)
}
