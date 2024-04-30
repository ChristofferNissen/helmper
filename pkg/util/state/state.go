package state

import (
	"github.com/spf13/viper"
)

func GetValue[T any](v *viper.Viper, key string) T {
	return v.Get(key).(T)
}

func SetValue[T any](v *viper.Viper, key string, value T) {
	v.Set(key, value)
}
