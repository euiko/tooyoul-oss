package miner

import (
	"context"
	"errors"

	"github.com/euiko/tooyoul/mineman/pkg/app/api"
)

var (
	ErrMinerAlreadyStarted = errors.New("miner already started")
	ErrMinerAlreadyStopped = errors.New("miner already stopped")
)

type Algorithm string

const (
	Undefined       Algorithm = ""
	Ethash          Algorithm = "ethash"     // eth
	Etchash         Algorithm = "etchash"    // etc
	Kawpow          Algorithm = "kawpow"     // raven
	Autolykos2      Algorithm = "autolykos2" // ergo
	Verthash        Algorithm = "verthash"   // ctx
	Nimiq           Algorithm = "nimiw"      // (nimiq)
	Lyra2z          Algorithm = "lyra2z"
	Phi2            Algorithm = "phi2" // (lux, argoneum)
	Lyra2rev3       Algorithm = "lyra2rev3"
	X16r            Algorithm = "x16r"
	X16rv2          Algorithm = "x16rv2"
	X16s            Algorithm = "x16s"             // (pgn, xsh)
	X16rt           Algorithm = "x16rt"            // (gin)
	Mtp             Algorithm = "mtp"              // (zcoin)
	Cuckatoo31Grin  Algorithm = "cuckatoo31_grin"  // (grin)
	Cuckarood29Grin Algorithm = "cuckarood29_grin" // (grin)
	Cnv8            Algorithm = "cnv8"
	Cnr             Algorithm = "cnr"          // (old monero)
	Cnv8Half        Algorithm = "cnv8_half"    // (stellite, masari)
	Cnv8Dbl         Algorithm = "cnv8_dbl"     // (x-cash)
	Cnv8Rwz         Algorithm = "cnv8_rwz"     // (graft)
	Cnv8Trtl        Algorithm = "cnv8_trtl"    // (old turtlecoin, loki)
	Cnv8Upx2        Algorithm = "cnv8_upx2"    // (uplexa)
	CnHeavy         Algorithm = "cn_heavy"     // (classic CN heavy)
	CnHaven         Algorithm = "cn_haven"     // (haven)
	CnSaber         Algorithm = "cn_saber"     // (bittube)
	CnConceal       Algorithm = "cn_conceal"   // (conceal)
	TrtlChukwa      Algorithm = "trtl_chukwa"  // (turtlecoin)
	TrtlChukwa2     Algorithm = "trtl_chukwa2" // (turtlecoin)
)

type (
	Downloadable interface {
		Download(ctx context.Context) <-chan error
	}

	Settings struct {
		Executor Executor
		Device   *DeviceQuery
		Pool     Pool
	}

	Option interface {
		Configure(o *Settings)
	}

	OptionFunc func(o *Settings)

	Miner interface {
		api.Module
		Name() string
		Algorithms() []Algorithm
		Start(ctx context.Context) error
		Stop() error
		Available() bool
	}

	MinerFactory func(*Settings) Miner
)

func (f OptionFunc) Configure(o *Settings) {
	f(o)
}

func WithExecutor(executor Executor) Option {
	return OptionFunc(func(o *Settings) {
		o.Executor = executor
	})
}

func WithPool(pool Pool) Option {
	return OptionFunc(func(o *Settings) {
		o.Pool = pool
	})
}

func WithDevice(device *DeviceQuery) Option {
	return OptionFunc(func(o *Settings) {
		o.Device = device
	})
}

func WithDeviceByIndex(index int) Option {
	return OptionFunc(func(o *Settings) {
		o.Device = &DeviceQuery{
			ByIndex: index,
			By:      ByIndex,
		}
	})
}

func WithDeviceByName(name string) Option {
	return OptionFunc(func(o *Settings) {
		o.Device = &DeviceQuery{
			ByName: name,
			By:     ByName,
		}
	})
}

func WithDeviceByBus(bus string) Option {
	return OptionFunc(func(o *Settings) {
		o.Device = &DeviceQuery{
			ByBus: bus,
			By:    ByBus,
		}
	})
}

func newSettings(opts ...Option) *Settings {
	// use default settings when nil
	settings := &Settings{
		Executor: NewPathExecutor(""),
		Pool: Pool{
			Algorithm: Undefined,
			Pass:      "x",
		},
		Device: nil,
	}

	for _, f := range opts {
		f.Configure(settings)
	}

	return settings
}
