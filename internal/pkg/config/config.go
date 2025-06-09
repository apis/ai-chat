package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"reflect"
	"strconv"
)

func parseConfigStruct(config any) {
	valueOf := reflect.ValueOf(config).Elem()
	structType := valueOf.Type()
	//structType := reflect.TypeOf(config)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		keyValue := field.Name
		defaultValue := field.Tag.Get("config_default")

		//if defaultValue == "" {
		//	log.Panic().Str("config_default", keyValue).Msg("default value is empty")
		//}

		description := field.Tag.Get("config_description")
		if description == "" {
			log.Panic().Str("config_description", keyValue).Msg("description value is empty")
		}

		kind := field.Type.Kind()
		switch kind {
		case reflect.Int:
			setIntItem(keyValue, defaultValue, description)
			break
		case reflect.String:
			setStringItem(keyValue, defaultValue, description)
			break
		default:
			log.Panic().Str("config_item", keyValue).Str("type", kind.String()).Msg("unsupported type")
		}
	}
}

// Parse configuration from command line, environment variables, config file
// Configuration options order precedence (from highest to lowest):
// - command line flag
// - environment variable
// - config file entry
// - default value
func Parse(config any, envPrefix string) {
	viper.SetEnvPrefix(envPrefix)
	configFileKey := "ConfigFile"
	weirdValue := "&^%"
	pflag.StringP(configFileKey, "c", weirdValue, "Config file location")
	pflag.Lookup(configFileKey).DefValue = ""

	parseConfigStruct(config)
	parseFlags()

	configFile := viper.GetString(configFileKey)
	if configFile != weirdValue {
		readConfig(configFile)
	}

	err := viper.Unmarshal(config)
	if err != nil {
		log.Fatal().Err(err).Msg("viper.Unmarshal() failed")
	}
}

func parseFlags() {
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatal().Err(err).Msg("viper.BindPFlags() failed")
	}
}
func readConfig(configFile string) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("viper.ReadInConfig() failed")
	}
}

func setIntItem(keyValue string, defaultValue string, description string) {
	intDefaultValue, err := strconv.Atoi(defaultValue)
	if err != nil {
		log.Panic().Err(err).Str("config_item", keyValue).Msg("strconv.Atoi() failed")
	}
	viper.SetDefault(keyValue, intDefaultValue)
	bindEnv(keyValue)
	pflag.Int(keyValue, intDefaultValue, description)
}

func setStringItem(keyValue string, defaultValue string, description string) {
	viper.SetDefault(keyValue, defaultValue)
	bindEnv(keyValue)
	pflag.String(keyValue, defaultValue, description)
}

func bindEnv(keyValue string) {
	err := viper.BindEnv(keyValue)
	if err != nil {
		log.Panic().Err(err).Str("config_item", keyValue).Msg("viper.BindEnv() failed")
	}
}
