package collectors

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	helix "github.com/nicklaw5/helix/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "twitch"
)

var (
	userTokenScopes = []string{"channel:read:subscriptions", "user:read:email"}
)

type TwitchChannel struct {
	Name        string
	ViewerCount int
	SubCount    int
}

type ApiSettings struct {
	Options           helix.Options
	appTokenIssuedAt  time.Time
	appTokenExpireIn  int
	AuthorizationURL  string
	AuthorizationCode string
	UserRefreshToken  string
}

type Settings struct {
	ApiSettings ApiSettings
	Channels    []TwitchChannel
	UserToken   bool
	User        TwitchChannel
	LogLevel    string
	LogFormat   string
	MetricsPath string
	ListenPort  string
	Address     string
}

type metrics struct {
	isLive			  *prometheus.Desc
	viewerCount		  *prometheus.Desc
	subCount		  *prometheus.Desc
	followerCount    *prometheus.Desc
}

type Exporter struct {
	client   *helix.Client
	metrics  *metrics
	Settings *Settings
	Logger   *slog.Logger
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.metrics.isLive
	ch <- e.metrics.viewerCount
	ch <- e.metrics.subCount
	ch <- e.metrics.followerCount
}

func (e *Exporter) handleAppTokens() {
	if e.isAppTokenExpired() {
		e.Logger.Info("Getting new application token")
		e.setNewAppToken()
	}
}

func (e *Exporter) handleUserTokens() {
	apiSettings := e.Settings.ApiSettings
	if apiSettings.UserRefreshToken == "" {
		e.Logger.Info("No refresh token available, generating new user access token")
		e.setNewUserToken()
		return
	}

	if !e.isUserTokenValid() {
		e.Logger.Info("User token no longer valid, refreshing")
		e.refreshUserToken()
	}
}

func (e *Exporter) collectUserMetrics() bool {
	if !e.Settings.UserToken {
		return false
	}

	if e.Settings.User.Name == "" {
		e.Logger.Warn("User token provided, but no user was provided, consider removing the --user.token flag or set a user to monitor. Not scraping user metrics")

		return false
	}

	return true
}

// TODO: Add Subs metrics
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	if e.Settings.UserToken {
		if e.Settings.ApiSettings.AuthorizationCode == "" {
			e.Logger.Error("User token provided, but authentication flow not completed, please use the authorization url")

			return
		}
		e.handleUserTokens()
	} else {
		e.handleAppTokens()
	}

	for _, twitchChannel := range e.Settings.Channels {

		ch <- prometheus.MustNewConstMetric(
			e.metrics.isLive,
			prometheus.GaugeValue,
			float64(e.IsLive(twitchChannel.Name)),
			twitchChannel.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			e.metrics.viewerCount,
			prometheus.GaugeValue,
			float64(e.ViewerCount(twitchChannel.Name)),
			twitchChannel.Name,
		)
	}

	if e.collectUserMetrics() {
		ch <- prometheus.MustNewConstMetric(
			e.metrics.subCount,
			prometheus.GaugeValue,
			float64(e.subCount()),
			e.Settings.User.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			e.metrics.followerCount,
			prometheus.GaugeValue,
			float64(e.followerCount()),
			e.Settings.User.Name,
		)
	}
}

// Returns 1 if broadcasting, 0 if not
func (e *Exporter) IsLive(channelName string) int {
	resp, err := e.client.SearchChannels(&helix.SearchChannelsParams{
		Channel: channelName,
	})
	if err != nil {
		e.Logger.Error("Failed to get channel status", "err", err)
		return 0
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get channel status", "statusCude", resp.StatusCode, "err", resp.ErrorMessage)
		return 0
	}

	for _, channel := range resp.Data.Channels {
		if strings.EqualFold(channel.DisplayName, channelName) && channel.IsLive {
			return 1
		}
	}

	return 0
}

func (e *Exporter) ViewerCount(channelName string) int {
	resp, err := e.client.GetStreams(&helix.StreamsParams{
		UserLogins: []string{channelName},
	})
	if err != nil {
		e.Logger.Error("Failed to get viewer count", "err", err)
		return 0
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get viewer count", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
		return 0
	}

	if len(resp.Data.Streams) <= 0 {
		return 0
	}

	return resp.Data.Streams[0].ViewerCount
}

func (e *Exporter) getUserID() string {
	resp, err := e.client.GetUsers(&helix.UsersParams{
		Logins: []string{e.Settings.User.Name},
	})
	if err != nil {
		e.Logger.Error("Failed to get user id", "err", err)
		return ""
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get user id", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
		return ""
	}

	var user helix.User
	for _, u := range resp.Data.Users {
		if u.Login == e.Settings.User.Name {
			user = u
		}
	}

	if user.ID == "" {
		e.Logger.Error(fmt.Sprintf("Could not find user with login %v", e.Settings.User))
		return ""
	}

	return user.ID
}

// TODO: Add more granularity on the metrics, by gifted and tier
func (e *Exporter) subCount() int {
	resp, err := e.client.GetSubscriptions(&helix.SubscriptionsParams{
		BroadcasterID: e.getUserID(),
		First:         1,
	})

	if err != nil {
		e.Logger.Error("Failed to get subscribers", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get subscribers", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	return len(resp.Data.Subscriptions)
}
 
func (e *Exporter) followerCount() int {
	resp, err := e.client.GetChannelFollows(&helix.GetChannelFollowsParams{
		BroadcasterID: e.getUserID(),
		First:         1,
	})

	if err != nil {
		e.Logger.Error("Failed to get followers", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get followers", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	return resp.Data.Total
}

func (e *Exporter) isUserTokenValid() bool {
	apiSettings := e.Settings.ApiSettings
	isValid, resp, err := e.client.ValidateToken(apiSettings.Options.UserAccessToken)
	if err != nil {
		e.Logger.Error("Failed to validate Token", "err", err)
		return false
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to validate Token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
		return false
	}

	return isValid
}

func (e *Exporter) isAppTokenExpired() bool {
	apiSettings := e.Settings.ApiSettings
	secondsSinceIssued := time.Since(apiSettings.appTokenIssuedAt).Seconds()
	thirtyMinutesInSeconds := 1800

	if apiSettings.appTokenExpireIn <= 0 {
		return true
	}

	// We need a new app token if its about to expire in less than 30 mins
	return int(secondsSinceIssued) >= (apiSettings.appTokenExpireIn - thirtyMinutesInSeconds)
}

func (e *Exporter) setNewAppToken() {
	resp, err := e.client.RequestAppAccessToken([]string{"user:read:email"})
	if err != nil {
		e.Logger.Error("Failed to request app access token", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to request app access token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	e.client.SetAppAccessToken(resp.Data.AccessToken)
	e.Settings.ApiSettings.appTokenExpireIn = resp.Data.ExpiresIn
	e.Settings.ApiSettings.appTokenIssuedAt = time.Now()
}

func (e *Exporter) setNewUserToken() {
	resp, err := e.client.RequestUserAccessToken(e.Settings.ApiSettings.AuthorizationCode)
	if err != nil {
		e.Logger.Error("Failed to request user access token", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to request user access token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	e.client.SetUserAccessToken(resp.Data.AccessToken)
	e.Settings.ApiSettings.UserRefreshToken = resp.Data.RefreshToken
}

func (e *Exporter) refreshUserToken() {
	resp, err := e.client.RefreshUserAccessToken(e.Settings.ApiSettings.UserRefreshToken)
	if err != nil {
		e.Logger.Error("Failed to refresh user access token", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to refresh user access token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	e.client.SetUserAccessToken(resp.Data.AccessToken)
	e.Settings.ApiSettings.UserRefreshToken = resp.Data.RefreshToken
}

func newMetrics() *metrics {
	return &metrics{
		isLive: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "is_live"),
			"If twitch channel is broadcasting",
			[]string{"name"}, nil,
		),

		viewerCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "viewer_total"),
			"Channel current viewer count",
			[]string{"name"}, nil,
		),

		subCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "subscribers_total"),
			"Channel current total subscribers",
			[]string{"name"}, nil,
		),

		followerCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "followers_total"),
			"Channel total number of followers",
			[]string{"name"}, nil,
		),
	}
}

func newHelixClient(s *Settings, logger *slog.Logger) (*helix.Client, error) {
	var client *helix.Client
	var err error

	if s.UserToken {
		client, err = helix.NewClient(&s.ApiSettings.Options)
		if err != nil {
			return nil, err
		}

		s.ApiSettings.AuthorizationURL = client.GetAuthorizationURL(&helix.AuthorizationURLParams{
			ResponseType: "code",
			Scopes:       userTokenScopes,
			State:        "prometheus-twitch-exporter",
			ForceVerify:  false,
		})
		logger.Info(fmt.Sprintf("Please authorize twitch exporter at: %v", s.ApiSettings.AuthorizationURL))

		return client, err
	}

	return helix.NewClient(&s.ApiSettings.Options)
}

func NewExporter(s *Settings, logger *slog.Logger) (*Exporter, error) {
	client, err := newHelixClient(s, logger)
	if err != nil {
		log.Fatalf("Failed to create twitch client %v", err)
	}

	metrics := newMetrics()

	exporter := &Exporter{
		client:   client,
		metrics:  metrics,
		Settings: s,
		Logger:   logger,
	}

	return exporter, nil
}
