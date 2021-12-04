package config

import "time"

type (

	// Value represent value of config fields
	Value interface {
		Bool(def ...bool) bool
		String(def ...string) string
		Float64(def ...float64) float64
		Duration(def ...time.Duration) time.Duration
		StringSlice(def ...[]string) []string
		StringMap(def ...map[string]interface{}) map[string]interface{}
		StringMapString(def ...map[string]string) map[string]string
		Scan(val interface{}) error
	}

	// OnChangedFunc for calling on config change callback
	OnChangedFunc func()

	Config interface {
		Sub(path string) Config
		Get(path string) Value
		Set(path string, val interface{}) error
		Scan(out interface{}) error
		Write() error
		OnChange(callback OnChangedFunc)
	}
)
