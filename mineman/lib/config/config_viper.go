package config

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type (
	ConfigViper struct {
		viper *viper.Viper
	}

	valueViper struct {
		key   string
		viper *viper.Viper
	}
)

func (c *ConfigViper) Get(path string) Value {
	return &valueViper{key: path, viper: c.viper}
}

func (c *ConfigViper) Set(path string, val interface{}) error {
	c.viper.Set(path, val)
	return nil
}

func (c *ConfigViper) Write() error {
	return c.viper.WriteConfig()
}

func (c *ConfigViper) OnChange(callback OnChangedFunc) {
	c.viper.OnConfigChange(func(in fsnotify.Event) {
		callback()
	})
}

func NewConfigViper(path string) *ConfigViper {
	v := viper.New()
	return &ConfigViper{
		viper: v,
	}
}

func (v *valueViper) Bool(def ...bool) bool {
	d := false
	if len(def) > 0 {
		d = def[0]
	}

	value := v.viper.Get(v.key)
	if val, ok := value.(bool); ok {
		return val
	}

	return d
}
func (v *valueViper) String(def ...string) string {
	d := ""
	if len(def) > 0 {
		d = def[0]
	}

	value := v.viper.Get(v.key)
	if val, ok := value.(string); ok {
		return val
	}

	return d

}
func (v *valueViper) Float64(def ...float64) float64 {
	d := 0.0
	if len(def) > 0 {
		d = def[0]
	}

	value := v.viper.Get(v.key)
	if val, ok := value.(float64); ok {
		return val
	}

	return d
}
func (v *valueViper) Duration(def ...time.Duration) time.Duration {
	d := time.Duration(0)
	if len(def) > 0 {
		d = def[0]
	}

	value := v.viper.Get(v.key)
	if val, ok := value.(time.Duration); ok {
		return val
	}

	return d
}
func (v *valueViper) StringSlice(def ...[]string) []string {
	d := []string{}
	if len(def) > 0 {
		d = def[0]
	}

	value := v.viper.Get(v.key)
	if val, ok := value.([]string); ok {
		return val
	}

	return d
}
func (v *valueViper) StringMap(def ...map[string]interface{}) map[string]interface{} {
	d := make(map[string]interface{})
	if len(def) > 0 {
		d = def[0]
	}

	if value, err := cast.ToStringMapE(v.viper.Get(v.key)); err == nil {
		return value
	}

	return d
}
func (v *valueViper) StringMapString(def ...map[string]string) map[string]string {
	d := make(map[string]string)
	if len(def) > 0 {
		d = def[0]
	}

	if value, err := cast.ToStringMapStringE(v.viper.Get(v.key)); err == nil {
		return value
	}

	return d
}

func (v *valueViper) Scan(val ...interface{}) error {
	return v.viper.UnmarshalKey(v.key, v.viper.Get(v.key))
}
