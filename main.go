package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	obsws "github.com/christopher-dG/go-obs-websocket"
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
	userAccessToken := viper.GetString("user_access_token")
	obsPort := viper.GetInt("obs_port")
	obsHostname := viper.GetString("obs_hostname")
	obsPassword := viper.GetString("obs_password")

	// Print our debugging information
	fmt.Println("-----------------------------------------------------")
	fmt.Println("Client ID: " + clientID)
	fmt.Println("Client Secret: " + clientSecret)
	fmt.Println("Redirect URL: " + redirectURL)
	fmt.Println("Channel Name: " + channelName)
	fmt.Println("User Access Token: " + userAccessToken)
	fmt.Println("-----------------------------------------------------")
	fmt.Println("Host & Port: " + obsHostname + ":" + strconv.Itoa(obsPort))

	// Setup timeout for requests for OBS
	obsws.SetReceiveTimeout(time.Second * 2)

	// Connect to OBS client
	c := obsws.Client{Host: obsHostname, Port: obsPort, Password: obsPassword}
	if err := c.Connect(); err != nil {
		log.Fatal(err)
	}
	defer c.Disconnect()

	// Authorize the app
	helixClient, err := helix.NewClient(&helix.Options{
		ClientID: clientID,
		//ClientSecret: clientSecret,
		UserAccessToken: userAccessToken,
		RedirectURI:     redirectURL,
	})
	if err != nil {
		panic(err)
	}

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

	// Print our debugging information
	fmt.Println("User ID: " + userID)
	fmt.Println("Channel ID: " + channelID)

	// Start listening to the PubSub API
	pubSubClient := pubsub.NewPool(userAccessToken, http.Header{})

	// Create the topic to listen to
	topic := fmt.Sprintf("channel-points-channel-v1.%s", channelID)

	// Listen to topic
	_, err = pubSubClient.Listen(topic, func(data pubsub.MessageData) {
		fmt.Println("-- PubSub Update ---------------------------------")

		// A reward has been redeemed cast to object
		var rewardRedeemed pubsub.RewardRedeemed
		json.Unmarshal([]byte(data.Message), &rewardRedeemed)

		// Grab the source name from our config file
		musicSource := viper.GetString("music_source")

		// Filter out unwanted titles that we don't need
		switch title := rewardRedeemed.Data.Redemption.Reward.Title; title {
		case pubsub.StatusMuteMusic:
			// Send and receive a request asynchronously.
			req := obsws.NewSetMuteRequest(musicSource, true)
			if err := req.Send(c); err != nil {
				log.Fatal(err)
			}
			// This will block until the response comes (potentially forever).
			resp, err := req.Receive()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Music has been set to: ", resp.Status())
		case pubsub.StatusUnMuteMusic:
			// Send and receive a request asynchronously.
			req := obsws.NewSetMuteRequest(musicSource, false)
			if err := req.Send(c); err != nil {
				log.Fatal(err)
			}
			// This will block until the response comes (potentially forever).
			resp, err := req.Receive()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Music has been set to: ", resp.Status())
		case pubsub.StatusSkipSong:
			// Command line arguments to use
			args := []string{"next"}

			// Execute the command to skip the next song
			cmd := exec.Command("mpc", args...)

			// Execute the command
			_, err = cmd.Output()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Song has been skipped")
		default:
			// Invalid title was passed
			fmt.Printf("Invalid redemption title \"%s\" was entered\n", title)
		}
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
