package miner

import (
	"strconv"
	"strings"
)

type DeviceSelectBy string

const (
	ByIndex DeviceSelectBy = "index"
	ByName  DeviceSelectBy = "name"
	ByBus   DeviceSelectBy = "bus"
)

type (
	DeviceQuery struct {
		ByIndex int
		ByName  string
		ByBus   string
		By      DeviceSelectBy
	}

	DeviceProvider interface {
		Select(query *DeviceQuery, target interface{}) (Device, error)
	}

	Device interface {
		Next() bool
		Scan(v ...interface{}) error
	}
)

func (q *DeviceQuery) String() string {
	if q == nil {
		return "<no-query>"
	}

	var d string

	switch q.By {
	case ByBus:
		d = q.ByBus
	case ByName:
		d = q.ByName
	default:
		d = strconv.Itoa(q.ByIndex)
	}

	return strings.Join([]string{
		string(q.By),
		d,
	}, ":")
}
