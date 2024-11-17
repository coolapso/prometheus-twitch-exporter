/*
Copyright © 2024 cool4pso
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/coolapso/prometheus-twitch-exporter/internal/collectors"
	"github.com/coolapso/prometheus-twitch-exporter/internal/httpServer"
	"github.com/coolapso/prometheus-twitch-exporter/internal/slogLogger"
	helix "github.com/nicklaw5/helix/v2"

	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "twitch-exporter",
	Short: "Exporter metrics from twitch to prometheus",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkCoreSettings()
	},

	Run: func(cmd *cobra.Command, args []string) {
		exporter()
	},
}

const (
	defaultLogLevel        = "info"
	defaultLogFormat       = "text"
	defaultMetricsPath     = "/metrics"
	defaultListenPort      = "9184"
	defaultAddress         = "localhost"
	defaultTwitchUserToken = false
	Version				   = "DEV"
)

var (
	settings       collectors.Settings
	twitchChannels []string
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	viper.AutomaticEnv()
	viper.SetDefault("LOG_LEVEL", defaultLogLevel)
	viper.SetDefault("LOG_FORMAT", defaultLogFormat)
	viper.SetDefault("METRICS_PATH", defaultMetricsPath)
	viper.SetDefault("LISTEN_PORT", defaultListenPort)
	viper.SetDefault("ADDRESS", defaultAddress)
	viper.SetDefault("TWITCH_CHANNELS", nil)
	viper.SetDefault("TWITCH_USER", "")
	viper.SetDefault("TWITCH_USER_TOKEN", defaultTwitchUserToken)
	viper.SetDefault("TWITCH_CLIENT_ID", "")
	viper.SetDefault("TWITCH_CLIENT_SECRET", "")
	viper.SetDefault("TWITCH_ACCESS_TOKEN", "")
	viper.SetDefault("TWITCH_REFRESH_TOKEN", "")

	rootCmd.Flags().StringVar(&settings.LogLevel, "log.level", defaultLogLevel, "Exporter log level")
	_ = viper.BindPFlag("log.level", rootCmd.Flags().Lookup("LOG_LEVEL"))

	rootCmd.Flags().StringVar(&settings.LogFormat, "log.format", defaultLogFormat, "Exporter log format, text or json")
	_ = viper.BindPFlag("log.format", rootCmd.Flags().Lookup("LOG_FORMAT"))

	rootCmd.Flags().StringVar(&settings.MetricsPath, "metrics.path", defaultMetricsPath, "Path to expose metrics at")
	_ = viper.BindPFlag("metrics.path", rootCmd.Flags().Lookup("METRICS_PATH"))

	rootCmd.Flags().StringVar(&settings.ListenPort, "listen.port", defaultListenPort, "Port to listen at")
	_ = viper.BindPFlag("listen.port", rootCmd.Flags().Lookup("LISTEN_PORT"))

	rootCmd.Flags().StringVar(&settings.Address, "address", defaultAddress, "The address to access the exporter used for oauth redirect uri")
	_ = viper.BindPFlag("address", rootCmd.Flags().Lookup("ADDRESS"))

	rootCmd.Flags().StringSliceVar(&twitchChannels, "twitch.channels", nil, "List of channels to get basic metrics from")
	_ = viper.BindPFlag("twitch.channels", rootCmd.Flags().Lookup("TWITCH_CHANNELS"))

	rootCmd.Flags().StringVar(&settings.User.Name, "twitch.user", "", "The user associated with the user token to get extra metrics from")
	_ = viper.BindPFlag("twitch.user", rootCmd.Flags().Lookup("TWITCH_USER"))

	rootCmd.Flags().BoolVar(&settings.UserToken, "user.token", false, "If going to use the provided token as a user token")
	_ = viper.BindPFlag("user.token", rootCmd.Flags().Lookup("TWITCH_USER_TOKEN"))

	rootCmd.Flags().StringVar(&settings.ApiSettings.Options.ClientID, "client.id", "", "twitch client id")
	_ = viper.BindPFlag("client.id", rootCmd.Flags().Lookup("TWITCH_CLIENT_ID"))

	rootCmd.Flags().StringVar(&settings.ApiSettings.Options.ClientSecret, "client.secret", "", "twitch client secret")
	_ = viper.BindPFlag("client.secret", rootCmd.Flags().Lookup("TWITCH_CLIENT_SECRET"))

	rootCmd.Flags().StringVar(&settings.ApiSettings.Options.ClientSecret, "access.token", "", "twitch user access token")
	_ = viper.BindPFlag("access.token", rootCmd.Flags().Lookup("TWITCH_ACCESS_TOKEN"))

	rootCmd.Flags().StringVar(&settings.ApiSettings.Options.ClientSecret, "refresh.token", "", "twitch refresh token")
	_ = viper.BindPFlag("refresh.token", rootCmd.Flags().Lookup("TWITCH_REFRESH_TOKEN"))

	settings.LogLevel = viper.GetString("LOG_LEVEL")
	settings.LogFormat = viper.GetString("LOG_FORMAT")
	settings.MetricsPath = viper.GetString("METRICS_PATH")
	settings.ListenPort = viper.GetString("LISTEN_PORT")
	settings.Address = viper.GetString("ADDRESS")
	twitchChannels = viper.GetStringSlice("TWITCH_CHANNELS")
	settings.User.Name = viper.GetString("TWITCH_USER")
	settings.UserToken = viper.GetBool("TWITCH_USER_TOKEN")
	settings.ApiSettings = collectors.ApiSettings{
		Options: helix.Options{
			ClientID:        viper.GetString("TWITCH_CLIENT_ID"),
			ClientSecret:    viper.GetString("TWITCH_CLIENT_SECRET"),
			UserAccessToken: viper.GetString("TWITCH_ACCESS_TOKEN"),
			RefreshToken:    viper.GetString("TWITCH_REFRESH_TOKEN"),
			RedirectURI:     fmt.Sprint("http://" + viper.GetString("ADDRESS") + ":" + viper.GetString("LISTEN_PORT")),
		},
	}
}

func checkCoreSettings() error {
	s := &settings
	if s.ApiSettings.Options.ClientID == "" {
		return fmt.Errorf("Missing client ID")
	}

	if s.ApiSettings.Options.ClientSecret == "" {
		return fmt.Errorf("Missing client secret")
	}

	return nil
}

// Set The twitch settings.TwitchChannel struct and append
// settings.user if not on the list, otherwise only user metrics will be collected
func setChannelList(s *collectors.Settings) {
	isInList := false
	for _, c := range twitchChannels {
		if s.User.Name == c {
			isInList = true
		}
		s.Channels = append(s.Channels, collectors.TwitchChannel{Name: c})
	}

	if !isInList && s.User.Name != "" {
		s.Channels = append(s.Channels, collectors.TwitchChannel{Name: s.User.Name})
	}
}

func exporter() {
	s := &settings
	setChannelList(s)

	logger, err := slogLogger.NewLogger(s.LogLevel, s.LogFormat)
	if err != nil {
		logger.Warn(err.Error())
	}

	if !s.UserToken {
		logger.Info("User token not enabled, using application token instead")
	}

	logger.Info(fmt.Sprintf("starting prometheus twitch exporter %v %v", Version, version.BuildContext()))
	exporter, err := collectors.NewExporter(s, logger)
	if err != nil {
		logger.Error("Failed to create new exporter", "err", err)
		os.Exit(1)
	}

	srv := httpServer.NewServer(exporter)
	logger.Info(fmt.Sprintf("Server ready and listening on port :%v", s.ListenPort))
	log.Fatal(srv.ListenAndServe())
}
