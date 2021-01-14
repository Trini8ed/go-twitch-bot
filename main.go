package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// Main program execution thread
func main() {

	// Read in from the config file
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	//Read in from our config file
	clientID := viper.GetString("client_id")

}
