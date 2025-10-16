package main

import (
	"strconv"

	"github.com/spf13/viper"
)

type config struct {
	dbName     string
	dbUser     string
	dbPassword string
	dbHost     string
	dbPort     int
	logging    *bool
	timming    *bool
	nodb       *bool
	root       *string
}

func defaults() {
	viper.Set("database.host", "localhost")
	viper.Set("database.port", 5432)
	viper.Set("database.user", "dbuser")
	viper.Set("database.password", "dbpassword")
	viper.Set("database.dbname", "dbname")

	viper.Set("config.debug", false)
	viper.Set("config.timming", false)
	viper.Set("config.root", ".")
	viper.Set("config.maxconnections", 10)
	viper.Set("config.nodb", false)

	viper.SafeWriteConfig()
}

func configure() config {
	viper.SetConfigName("sha3sum")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("$HOME/.config/")

	err := viper.ReadInConfig()
	if err != nil {
		defaults()
	}

	cfg := config{}
	cfg.dbHost = viper.GetString("database.host")
	cfg.dbPort, err = strconv.Atoi(viper.GetString("database.port"))
	if err != nil {
		cfg.dbPort = 5432
	}
	cfg.dbUser = viper.GetString("database.user")
	cfg.dbPassword = viper.GetString("database.password")
	cfg.dbName = viper.GetString("database.dbname")

	cfg.logging = new(bool)
	cfg.timming = new(bool)
	cfg.nodb = new(bool)
	cfg.root = new(string)
	*cfg.logging = viper.GetBool("config.debug")
	*cfg.timming = viper.GetBool("config.timming")
	*cfg.nodb = viper.GetBool("config.nodb")
	*cfg.root = viper.GetString("config.root")

	return cfg
}
