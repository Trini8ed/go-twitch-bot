package pubsub

import "time"

// RewardRedeemed is the structure when a redemption point is redeemed
type RewardRedeemed struct {
	Type string `json:"type"`
	Data struct {
		Timestamp  time.Time `json:"timestamp"`
		Redemption struct {
			ID   string `json:"id"`
			User struct {
				ID          string `json:"id"`
				Login       string `json:"login"`
				DisplayName string `json:"display_name"`
			} `json:"user"`
			ChannelID  string    `json:"channel_id"`
			RedeemedAt time.Time `json:"redeemed_at"`
			Reward     struct {
				ID                  string      `json:"id"`
				ChannelID           string      `json:"channel_id"`
				Title               string      `json:"title"`
				Prompt              string      `json:"prompt"`
				Cost                int         `json:"cost"`
				IsUserInputRequired bool        `json:"is_user_input_required"`
				IsSubOnly           bool        `json:"is_sub_only"`
				Image               interface{} `json:"image"`
				DefaultImage        struct {
					URL1X string `json:"url_1x"`
					URL2X string `json:"url_2x"`
					URL4X string `json:"url_4x"`
				} `json:"default_image"`
				BackgroundColor string `json:"background_color"`
				IsEnabled       bool   `json:"is_enabled"`
				IsPaused        bool   `json:"is_paused"`
				IsInStock       bool   `json:"is_in_stock"`
				MaxPerStream    struct {
					IsEnabled    bool `json:"is_enabled"`
					MaxPerStream int  `json:"max_per_stream"`
				} `json:"max_per_stream"`
				ShouldRedemptionsSkipRequestQueue bool        `json:"should_redemptions_skip_request_queue"`
				TemplateID                        interface{} `json:"template_id"`
				UpdatedForIndicatorAt             time.Time   `json:"updated_for_indicator_at"`
				MaxPerUserPerStream               struct {
					IsEnabled           bool `json:"is_enabled"`
					MaxPerUserPerStream int  `json:"max_per_user_per_stream"`
				} `json:"max_per_user_per_stream"`
				GlobalCooldown struct {
					IsEnabled             bool `json:"is_enabled"`
					GlobalCooldownSeconds int  `json:"global_cooldown_seconds"`
				} `json:"global_cooldown"`
				RedemptionsRedeemedCurrentStream interface{} `json:"redemptions_redeemed_current_stream"`
				CooldownExpiresAt                interface{} `json:"cooldown_expires_at"`
			} `json:"reward"`
			Status string `json:"status"`
		} `json:"redemption"`
	} `json:"data"`
}

// Constants related to the Twitch Bot
const (
	StatusUnMuteMusic = "Turn on the music B)" // Status indicating that OBS needs to mute the music
	StatusMuteMusic   = "MUTE THE MUSIC"       // Status indicating that OBS needs to un-mute the music
	StatusSkipSong    = "Skip song"            // Status indicating to execute CMD mdb next
)
