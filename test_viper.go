package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Postgres struct {
		Host string `mapstructure:"host"`
	} `mapstructure:"postgres"`
}

func main() {
	v := viper.New()
	v.SetConfigFile("test.yaml")
	os.WriteFile("test.yaml", []byte("postgres:\n  host: localhost\n"), 0644)
	
	v.BindEnv("postgres.host", "DB_HOST")
	os.Setenv("DB_HOST", "db_container")
	
	v.ReadInConfig()
	
	var c Config
	v.Unmarshal(&c)
	fmt.Printf("Host is: %s\n", c.Postgres.Host)
}
