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
	respAppToken, err := helixClient.RequestAppAccessToken([]string{"channel:moderate"})
	if err != nil {
		panic(err)
	}

	// Save the app access token
	appToken := respAppToken.Data.AccessToken

	fmt.Println("-- App Token ----------------------------------------")
	fmt.Println(respAppToken.Data)

	// Set the access token on the client
	helixClient.SetAppAccessToken(appToken)

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

	// Get the user id and channel id
	userID := respUser.Data.Users[0].ID
	channelID := respUser.Data.Users[0].ID

	// Start listening to the PubSub API
	pubSubClient := pubsub.NewPool(appToken, http.Header{})

	// Create the topic to listen to
	topic := fmt.Sprintf("chat_moderator_actions.%v.%v", userID, channelID)

	// Listen to topic
	_, err = pubSubClient.Listen(topic, func(data pubsub.MessageData) {
		fmt.Printf("Moderator action: %s\n", data.Message)
	})
	if err != nil {
		panic(err)
	}

	// Function callback for when we start our PubSub client
	pubSubClient.OnStart = func() {
		fmt.Println("-- OnStart --------------------------------------")
		fmt.Println("Starting Pub Sub!")
	}

	// Function callback for when we get an error on our PubSub client
	pubSubClient.OnError = func(psc *pubsub.Conn, e error, i interface{}) {
		fmt.Println("-- OnError --------------------------------------")
		fmt.Println("Error has occurred")
		fmt.Println(psc)
		fmt.Println(i)
		panic(e)
	}

	// Function callback for when our PubSub client connects to Twitch API
	pubSubClient.OnConnect = func(conn *pubsub.Conn) {
		fmt.Println("-- OnConnect ------------------------------------")
		fmt.Println("Connected to Twitch API")
	}

	// Start and wait
	err = pubSubClient.Start()
	if err != nil {
		panic(err)
	}

	// Block the main thread so we keep listening to topics
	wait := make(chan bool)
	<-wait

}
