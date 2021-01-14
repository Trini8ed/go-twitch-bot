package main

import (
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix"
	"github.com/spf13/viper"
	"github.com/trini8ed/go-twitch-bot/pubsub"
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

	// Read in from our config file
	clientID := viper.GetString("client_id")
	clientSecret := viper.GetString("client_secret")
	redirectURL := viper.GetString("redirect_url")
	channelName := viper.GetString("channel_name")

	fmt.Println("-----------------------------------------------------")
	fmt.Println("Client ID: " + clientID)
	fmt.Println("Client Secret: " + clientSecret)
	fmt.Println("Redirect URL: " + redirectURL)
	fmt.Println("Channel Name: " + channelName)

	// Authorize the app
	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURL,
	})
	if err != nil {
		panic(err)
	}

	// Get the app access token with specified parameters
	respAppToken, err := helixClient.RequestAppAccessToken([]string{"channel:read:redemptions"})
	if err != nil {
		panic(err)
	}

	fmt.Println("-- App Token ----------------------------------------")
	fmt.Println(respAppToken.Data)

	// Set the access token on the client
	helixClient.SetAppAccessToken(respAppToken.Data.AccessToken)

	// Get the channel id of the user
	respUser, err := helixClient.GetUsers(&helix.UsersParams{
		Logins: []string{channelName},
	})
	if err != nil {
		panic(err)
	}
	if len(respUser.Data.Users) == 0 {
		panic(fmt.Errorf("No user was found under the name %v", channelName))
	}

	fmt.Println("-- Get Users ----------------------------------------")
	fmt.Println(respUser.Data)

	// Start listening to the PubSub API
	pubSubClient := pubsub.NewPubSubPool(respAppToken.Data.AccessToken, http.Header{})

	// Create the topic to listen to
	topic := fmt.Sprintf("channel-points-channel-v1.%v", respUser.Data.Users[0])

	// Start listening to the messages
	_, err = pubSubClient.Listen(topic, func(data pubsub.MessageData) {
		fmt.Printf(data.Message)
	})
	if err != nil {
		panic(err)
	}

	for {
		// Keep listening for changes forever
	}

}
