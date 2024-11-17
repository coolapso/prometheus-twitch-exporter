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
	userTokenScopes = []string{"channel:read:subscriptions"}
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
	isLive        *prometheus.Desc
	viewerCount   *prometheus.Desc
	subCount      *prometheus.Desc
	followerCount *prometheus.Desc
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
		e.Logger.Info("getting new application token")
		e.setNewAppToken()
	}
}

func (e *Exporter) handleUserTokens() error {
	apiSettings := e.Settings.ApiSettings

	if apiSettings.Options.UserAccessToken == "" {
		if apiSettings.AuthorizationCode == "" {
			return fmt.Errorf("Authentication flow not completed, please use the authorization url to obtain an access token, or provide an access token")
		}

		e.setNewUserToken()
		return nil
	}

	if !e.isUserTokenValid() {
		if apiSettings.Options.RefreshToken == "" {
			return fmt.Errorf("Access token available, but no refresh token provided, Please re-authenticate again, or provide a refresh token")
		}

		e.Logger.Info("User token no longer valid, refreshing")
		e.refreshUserToken()
		return nil
	}

	e.Logger.Debug("token is valid")
	return nil
}

func (e *Exporter) collectUserMetrics() bool {
	if e.Settings.UserToken {
		if e.Settings.User.Name == "" {
			e.Logger.Warn("User token provided, but no user was provided, consider removing the --user.token flag or set a user to monitor. Not scraping user metrics")

			return false
		}

		return true
	}

	return false
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	if e.Settings.UserToken {
		err := e.handleUserTokens()
		if err != nil {
			e.Logger.Error(err.Error())
			return
		}
	} else {
		e.handleAppTokens()
	}

	for _, twitchChannel := range e.Settings.Channels {

		ch <- prometheus.MustNewConstMetric(
			e.metrics.isLive,
			prometheus.GaugeValue,
			float64(e.isLive(twitchChannel.Name)),
			twitchChannel.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			e.metrics.viewerCount,
			prometheus.GaugeValue,
			float64(e.viewerCount(twitchChannel.Name)),
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
func (e *Exporter) isLive(channelName string) int {
	e.Logger.Debug("getting channel status", "channelName", channelName)
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

func (e *Exporter) viewerCount(channelName string) int {
	e.Logger.Debug("getting viewer count", "channelName", channelName)
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

	vc := resp.Data.Streams[0].ViewerCount
	e.Logger.Debug("Got channel viewer count", "channelName", channelName, "count", vc)
	return vc
}

func (e *Exporter) getUserID() string {
	e.Logger.Debug("getting user ID", "user", e.Settings.User.Name)
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
	userID := user.ID

	if userID == "" {
		e.Logger.Error(fmt.Sprintf("Could not find user with login %v", e.Settings.User))
		return ""
	}

	e.Logger.Debug("user ID found", "userID", userID)
	return userID
}

// TODO: Add more granularity on the metrics, by gifted and tier
func (e *Exporter) subCount() int {
	e.Logger.Debug("getting user sub count")
	resp, err := e.client.GetSubscriptions(&helix.SubscriptionsParams{
		BroadcasterID: e.getUserID(),
	})

	if err != nil {
		e.Logger.Error("Failed to get subscribers", "err", err)
		return 0
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to get subscribers", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
		return 0
	}

	sc := len(resp.Data.Subscriptions)
	e.Logger.Debug("got subcount", "subCount", sc)
	return sc
}

func (e *Exporter) followerCount() int {
	e.Logger.Debug("getting user follower count")
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

	fc := resp.Data.Total
	e.Logger.Debug("got channel follower count", "channelName", e.Settings.User.Name, "count", fc)

	return resp.Data.Total
}

func (e *Exporter) isUserTokenValid() bool {
	e.Logger.Debug("validating user token")
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
	e.Logger.Debug("checking if application token is expired")
	apiSettings := e.Settings.ApiSettings
	secondsSinceIssued := time.Since(apiSettings.appTokenIssuedAt).Seconds()
	thirtyMinutesInSeconds := 1800

	if apiSettings.appTokenExpireIn <= 0 {
		return true
	}

	return int(secondsSinceIssued) >= (apiSettings.appTokenExpireIn - thirtyMinutesInSeconds)
}

func (e *Exporter) setNewAppToken() {
	e.Logger.Debug("setting new application token")
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
	e.Logger.Debug("new application token set")
}

func (e *Exporter) setNewUserToken() {
	e.Logger.Debug("setting new user token")
	resp, err := e.client.RequestUserAccessToken(e.Settings.ApiSettings.AuthorizationCode)
	if err != nil {
		e.Logger.Error("Failed to request user access token", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to request user access token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	e.client.SetUserAccessToken(resp.Data.AccessToken)
	e.Settings.ApiSettings.Options.RefreshToken = resp.Data.RefreshToken
	e.Logger.Debug("new user token set")
}

func (e *Exporter) refreshUserToken() {
	e.Logger.Debug("refreshing user token")
	resp, err := e.client.RefreshUserAccessToken(e.Settings.ApiSettings.Options.RefreshToken)
	if err != nil {
		e.Logger.Error("Failed to refresh user access token", "err", err)
	}

	if resp.StatusCode != 200 {
		e.Logger.Error("Failed to refresh user access token", "statusCode", resp.StatusCode, "err", resp.ErrorMessage)
	}

	e.client.SetUserAccessToken(resp.Data.AccessToken)
	e.Settings.ApiSettings.Options.RefreshToken = resp.Data.RefreshToken
	e.Logger.Debug("user token refreshed")
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
		logger.Info(fmt.Sprintf("twitch authorization url: %v", s.ApiSettings.AuthorizationURL))

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
