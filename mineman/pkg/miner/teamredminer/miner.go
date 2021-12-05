package teamredminer

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/config"
	pkgio "github.com/euiko/tooyoul/mineman/pkg/io"
	"github.com/euiko/tooyoul/mineman/pkg/log"
	"github.com/euiko/tooyoul/mineman/pkg/miner"
)

var (
	ErrCommandBufferFull = errors.New("command buffer is full")
)

const (
	name        = "teamredminer"
	execName    = name
	skipResult  = 5
	cmdBuffer   = 10
	waitTimeout = time.Second * 15
)

type (
	Config struct {
		URL      string `mapstructure:"url"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	}

	Miner struct {
		c      config.Config
		option miner.Option
		config Config

		ctx     context.Context
		cancel  func()
		cmdChan chan command

		// some variables that only accessible from the loop
		state      state
		stdOut     io.ReadCloser
		stdErr     io.ReadCloser
		stdIn      io.WriteCloser
		reader     *pkgio.ManagedReader
		execCancel func() // to cancel program prior to stopping
	}
)

func (m *Miner) Init(ctx context.Context, c config.Config) error {

	m.cmdChan = make(chan command, cmdBuffer)
	if err := c.Scan(&m.config); err != nil {
		return err
	}

	return nil
}

func (m *Miner) Close(ctx context.Context) error {
	cmd := newCommandStop()
	defer close(cmd.errChan)

	if err := m.do(cmd); err != nil {
		return err
	}
	if err := <-cmd.errChan; err != nil {
		return err
	}

	m.cancel()
	m.cancel = nil
	m.ctx = nil
	return nil
}

func (m *Miner) Configure(opts ...miner.OptionConfigurable) {
	// default executor
	m.option.Executor = miner.NewPathExecutor("")
	miner.LoadMinerOption(&m.option, opts...)
}

func (m *Miner) Algorithms() []miner.Algorithm {
	return []miner.Algorithm{
		miner.Ethash,
		miner.Etchash,
		miner.Kawpow,
		miner.Autolykos2,
		miner.Verthash,
		miner.Nimiq,
		miner.Lyra2z,
		miner.Phi2,
		miner.Lyra2rev3,
		miner.X16r,
		miner.X16rv2,
		miner.X16s,
		miner.X16rt,
		miner.Mtp,
		miner.Cuckatoo31Grin,
		miner.Cuckarood29Grin,
		miner.Cnv8,
		miner.Cnr,
		miner.Cnv8Half,
		miner.Cnv8Dbl,
		miner.Cnv8Rwz,
		miner.Cnv8Trtl,
		miner.Cnv8Upx2,
		miner.CnHeavy,
		miner.CnHaven,
		miner.CnSaber,
		miner.CnConceal,
		miner.TrtlChukwa,
		miner.TrtlChukwa2,
	}
}

func (m *Miner) Start(ctx context.Context) error {

	// start the goroutine
	if m.ctx == nil {
		log.Debug("starting background loop")
		m.ctx, m.cancel = context.WithCancel(ctx)
		go m.run(ctx)
	}

	log.Debug("sending start command")
	cmd := newCommandStart()
	defer close(cmd.errChan)

	if err := m.do(cmd); err != nil {
		return err
	}

	return <-cmd.errChan
}

func (m *Miner) Stop() error {

	cmd := newCommandStop()
	defer close(cmd.errChan)

	if err := m.do(cmd); err != nil {
		return err
	}

	return <-cmd.errChan
}

func (m *Miner) Select(query *miner.DeviceQuery, target interface{}) (miner.Device, error) {
	var result []gpuDevice

	log.Debug("start looking up for selected device with query=%s", log.WithValues(query.String()))
	cmd := m.option.Executor.Execute(context.Background(), execName, []string{"--list_devices"})

	b, err := cmd.Output()
	if err != nil {
		log.Debug("error occurred when collect teamredminer list_devices output", log.WithError(err))
		return nil, err
	}

	// scan through the result
	log.Debug("parsing gpu texts")
	gpus, err := parseDevices(b)
	if err != nil {
		return nil, err
	}
	log.Debug("found %d gpus", log.WithValues(len(gpus)))

	for _, gpu := range gpus {
		if query == nil {
			result = append(result, gpu)
			continue
		}

		var found bool
		// query by query type
		switch query.By {
		case miner.ByIndex:
			found = gpu.index == query.ByIndex
		case miner.ByName:
			found = strings.Contains(gpu.model, query.ByName)
		case miner.ByBus:
			found = gpu.busId == query.ByBus
		}

		if !found {
			continue
		}
		result = append(result, gpu)
	}
	log.Debug("found %d selected device result", log.WithValues(len(result)))

	return newDevice(result), nil
}

func (m *Miner) do(cmd command) error {
	select {
	case m.cmdChan <- cmd:
		return nil
	default:
		return ErrCommandBufferFull
	}
}

func (m *Miner) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			cmd := newCommandStop()
			m.stop(context.Background(), cmd)
			close(cmd.errChan)
			return
		case cmd := <-m.cmdChan:
			m.handleCmd(ctx, cmd)
		}
	}
}

func (m *Miner) handleCmd(ctx context.Context, cmd command) {
	var err error
	switch cmd := cmd.(type) {
	case *commandStart:
		err = m.start(ctx, cmd)
	case *commandStop:
		err = m.stop(ctx, cmd)
	}
	cmd.SendErr(err)
}

func (m *Miner) start(ctx context.Context, cmd *commandStart) error {
	if m.state == stateStarted {
		return miner.ErrMinerAlreadyStarted
	}

	log.Debug("building command and arguments")
	args, err := BuildCommandArgs(m)
	if err != nil {
		return err
	}

	// use wrapped context, so program initialize can be canceled in error case
	ctx, m.execCancel = context.WithCancel(ctx)

	log.Debug("get command execution")
	// bind std in/out and start the command
	execCmd := m.option.Executor.Execute(ctx, execName, args)
	cancelStart := func(stop bool) {
		m.stdIn = nil
		m.stdOut = nil
		m.stdErr = nil

		if stop {
			m.execCancel()
		}
	}

	log.Debug("piping std in/out and start command")
	m.stdIn, err = execCmd.StdinPipe()
	if err != nil {
		return err
	}

	m.stdOut, err = execCmd.StdoutPipe()
	if err != nil {
		return err
	}

	m.stdErr, err = execCmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := execCmd.Start(); err != nil {
		cancelStart(false)
		return err
	}

	log.Debug("starting manager to process stdout")
	m.reader = pkgio.NewManagedReader(m.stdOut, m.stdErr)
	if err := m.reader.StartAndWait(ctx, "Successfully initialized", waitTimeout); err != nil {
		cancelStart(true)
		return err
	}

	log.Info("teamredminer started",
		log.WithField("algorithm", m.option.Algorithm),
		log.WithField("device", m.option.Device.String()),
		log.WithField("url", m.config.URL),
		log.WithField("user", m.config.User),
	)

	m.state = stateStarted
	return nil
}

func (m *Miner) stop(ctx context.Context, cmd *commandStop) error {
	if m.state == stateStopped {
		return miner.ErrMinerAlreadyStopped
	}

	log.Debug("stopping reader and close stdin/out")
	if err := m.reader.Close(); err != nil {
		return err
	}

	log.Debug("closing stdin")
	if err := m.stdIn.Close(); err != nil {
		return err
	}

	m.execCancel()

	m.stdIn = nil
	m.stdOut = nil
	m.stdErr = nil

	m.state = stateStopped
	return nil
}

func New() *Miner {
	return &Miner{}
}

func newMiner() miner.Miner {
	return New()
}

func init() {
	miner.Register(name, newMiner)
}
