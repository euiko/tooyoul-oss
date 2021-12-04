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

	Option struct {
		Executor  Executor
		Algorithm Algorithm
		Device    *DeviceQuery
	}

	OptionConfigurable interface {
		Configure(o *Option)
	}

	OptionFunc func(o *Option)

	Miner interface {
		api.Module
		Configure(opts ...OptionConfigurable)
		Algorithms() []Algorithm
		Start(ctx context.Context) error
		Stop() error
	}

	MinerFactory func() Miner
)

func (f OptionFunc) Configure(o *Option) {
	f(o)
}

func LoadMinerOption(option *Option, opts ...OptionConfigurable) *Option {
	if option == nil {
		// use default option when nil
		option = &Option{
			Executor:  NewPathExecutor(""),
			Algorithm: Undefined,
			Device:    nil,
		}
	}

	for _, f := range opts {
		f.Configure(option)
	}

	return option
}

func WithExecutor(executor Executor) OptionConfigurable {
	return OptionFunc(func(o *Option) {
		o.Executor = executor
	})
}

func WithAlgorithm(algorithm Algorithm) OptionConfigurable {
	return OptionFunc(func(o *Option) {
		o.Algorithm = algorithm
	})
}

func WithDeviceByIndex(index int) OptionConfigurable {
	return OptionFunc(func(o *Option) {
		o.Device = &DeviceQuery{
			ByIndex: index,
			By:      ByIndex,
		}
	})
}

func WithDeviceByName(name string) OptionConfigurable {
	return OptionFunc(func(o *Option) {
		o.Device = &DeviceQuery{
			ByName: name,
			By:     ByName,
		}
	})
}

func WithDeviceByBus(bus string) OptionConfigurable {
	return OptionFunc(func(o *Option) {
		o.Device = &DeviceQuery{
			ByBus: bus,
			By:    ByBus,
		}
	})
}
