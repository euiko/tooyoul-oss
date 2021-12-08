package miner

import (
	"errors"
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

func (b DeviceSelectBy) Valid() bool {
	switch b {
	case ByIndex, ByName, ByBus:
		return true
	default:
		return false
	}
}

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

func ParseDeviceQuery(queryStr string) (*DeviceQuery, error) {
	splitted := strings.Split(strings.TrimSpace(queryStr), ":")
	if len(splitted) != 2 {
		return nil, errors.New("parse device query failed, must be in format <select_by>:<your_query>, e.g. index:1, name:RX 580 ")
	}

	by := DeviceSelectBy(strings.ToLower(splitted[0]))
	if !by.Valid() {
		return nil, errors.New("parse device query select by doesn't valid, it must be either of, index, bus, or name")
	}

	query := new(DeviceQuery)
	query.By = by

	queryExpr := strings.TrimSpace(splitted[1])
	switch by {
	case ByIndex:
		id, err := strconv.Atoi(queryExpr)
		if err != nil {
			return nil, err
		}
		query.ByIndex = id
	case ByBus:
		query.ByBus = queryExpr
	case ByName:
		query.ByName = queryExpr
	}

	return query, nil
}
