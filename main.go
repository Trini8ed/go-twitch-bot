package main

import (
	"fmt"

	"github.com/nicklaw5/helix"
	"github.com/spf13/viper"
)

// Main program execution thread
func main() {

	// Setup viper to read in the config file
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	// Obtain the client ID and secret ID from the config file
	clientID := viper.GetString("client_id")
	clientSecret := viper.GetString("client_secret")
	//userToken := viper.GetString("user_token")
	//appToken := viper.GetString("app_token")

	// Sign into wrapper using Twitch Client ID
	client, err := helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		//UserAccessToken: userToken,
		//UserAccessToken: userToken,
		//AppAccessToken:  appToken,
	})
	if err != nil {
		// There was an issue logging into out account
		panic(err)
	}

	resp, err := client.GetWebhookSubscriptions(&helix.WebhookSubscriptionsParams{
		First: 10,
	})
	if err != nil {
		// handle error
	}

	fmt.Printf("%+v\n", resp)

}
