package teamredminer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/euiko/tooyoul/mineman/pkg/log"
)

type (
	// device is internally used by teamredminer implementation for providing
	// miner.Miner interface
	device struct {
		index   int
		devices []gpuDevice
	}

	// gpuDevice is being used by internal implementation for holding
	// teamredminer's gpu data
	gpuDevice struct {
		index    int
		platform int
		opencl   int
		busId    string
		name     string
		model    string
		cu       int
	}
)

var parseDeviceRegex = regexp.MustCompile("\\t+")

func (d *device) Next() bool {
	if d.index >= len(d.devices) {
		return false
	}

	d.index++
	return true
}

func (d *device) Scan(values ...interface{}) error {
	// for teamredminer scan will be ordered from index to cu,
	// read the following gpuDevice structure

	if d.index >= len(d.devices) {
		return errors.New("end of result, cannot do more scan devices")
	}

	current := d.devices[d.index]
	vLen := len(values)

	// scan index if exists
	if vLen > 0 {
		if err := d.scanInt("index", current.index, values[0]); err != nil {
			return err
		}
	}

	// scan platform
	if vLen > 1 {
		if err := d.scanInt("platform", current.platform, values[1]); err != nil {
			return err
		}
	}

	// scan opencl
	if vLen > 2 {
		if err := d.scanInt("opencl", current.opencl, values[2]); err != nil {
			return err
		}
	}

	// scan busId
	if vLen > 3 {
		if err := d.scanString("busId", current.busId, values[3]); err != nil {
			return err
		}
	}

	// scan name
	if vLen > 4 {
		if err := d.scanString("name", current.name, values[4]); err != nil {
			return err
		}
	}

	// scan model
	if vLen > 5 {
		if err := d.scanString("model", current.model, values[5]); err != nil {
			return err
		}
	}

	// scan cu
	if vLen > 6 {
		if err := d.scanInt("cu", current.cu, values[6]); err != nil {
			return err
		}
	}

	return nil
}

func (d *device) scanInt(key string, value int, target interface{}) error {
	v, ok := target.(*int)
	if !ok {
		return fmt.Errorf("cannot scan key %s type of int to %t", key, target)
	}
	*v = value

	return nil
}

func (d *device) scanString(key string, value string, target interface{}) error {
	v, ok := target.(*string)
	if !ok {
		return fmt.Errorf("cannot scan key %s type of sring to %t", key, target)
	}
	*v = value

	return nil
}

func newDevice(devices []gpuDevice) *device {
	return &device{
		index:   -1,
		devices: devices,
	}
}

func parseDevices(gpuTexts []string) ([]gpuDevice, error) {
	var err error
	gpus := make([]gpuDevice, len(gpuTexts))
	for i, t := range gpuTexts {
		cols := parseDeviceRegex.Split(t, -1)
		if len(cols) != 7 {
			log.Debug("invalid column count after parse, expect 7, got %d", log.WithValues(len(cols)))
			continue
		}

		gpus[i].index, err = strconv.Atoi(cols[0])
		if err != nil {
			log.Debug("parse device index failed", log.WithError(err))
			continue
		}

		gpus[i].platform, err = strconv.Atoi(cols[1])
		if err != nil {
			log.Debug("parse device platform failed", log.WithError(err))
			continue
		}

		gpus[i].opencl, err = strconv.Atoi(cols[2])
		if err != nil {
			log.Debug("parse device opencl failed", log.WithError(err))
			continue
		}

		gpus[i].busId = cols[3]
		gpus[i].name = cols[4]
		gpus[i].model = cols[5]

		gpus[i].cu, err = strconv.Atoi(cols[6])
		if err != nil {
			log.Debug("parse device cu failed", log.WithError(err))
			continue
		}
	}
	return gpus, nil
}
