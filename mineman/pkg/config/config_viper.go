package config

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type (
	Viper struct {
		viper *viper.Viper

		// some options
		standalone bool
	}

	valueViper struct {
		key   string
		viper *viper.Viper
	}

	ViperOptions interface {
		Configure(v *Viper)
	}

	ViperOptionsFunc func(v *Viper)
)

func (f ViperOptionsFunc) Configure(v *Viper) {
	f(v)
}

func (c *Viper) Sub(path string) Config {
	return &Viper{
		viper: c.viper.Sub(path),
	}
}

func (c *Viper) Get(path string) Value {
	return &valueViper{key: path, viper: c.viper}
}

func (c *Viper) Set(path string, val interface{}) error {
	c.viper.Set(path, val)
	return nil
}

func (c *Viper) Scan(out interface{}) error {
	v := new(valueViper)
	v.key = ""
	v.viper = c.viper

	return v.Scan(out)
}

func (c *Viper) Write() error {
	return c.viper.WriteConfig()
}

func (c *Viper) OnChange(callback OnChangedFunc) {
	c.viper.OnConfigChange(func(in fsnotify.Event) {
		callback()
	})
}

func ViperStandalone() ViperOptions {
	return ViperOptionsFunc(func(v *Viper) {
		v.standalone = true
	})
}

func NewViper(path string, opts ...ViperOptions) *Viper {
	v := viper.New()
	vpr := Viper{
		viper: v,
	}

	for _, o := range opts {
		o.Configure(&vpr)
	}

	v.SetConfigName(path)
	v.AddConfigPath(".")

	if !vpr.standalone {
		if err := v.ReadInConfig(); err != nil {
			panic(err)
		}
	}

	return &vpr
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

func (v *valueViper) Scan(val interface{}) error {
	if v.key == "" {
		return v.viper.Unmarshal(val)
	}

	return v.viper.UnmarshalKey(v.key, val)
}
