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

type subscribers struct {
	total int
	t1    int
	t2    int
	t3    int
	prime int
}

type TwitchChannel struct {
	Name        string
	ViewerCount int
	Subscribers subscribers
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
	User        string
	LogLevel    string
	LogFormat   string
	MetricsPath string
	ListenPort  string
	Address     string
}

type metrics struct {
	isLive      *prometheus.Desc
	viewerCount *prometheus.Desc
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

	if e.Settings.User == "" {
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
		isLive, err := e.IsLive(twitchChannel.Name)
		if err != nil {
			e.Logger.Error("Failed to get channel status", "err", err)
		}

		viewerCount, err := e.ViewerCount(twitchChannel.Name)
		if err != nil {
			e.Logger.Error("Failed to get viewer count", "err", err)
		}

		ch <- prometheus.MustNewConstMetric(
			e.metrics.isLive,
			prometheus.GaugeValue,
			float64(isLive),
			twitchChannel.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			e.metrics.viewerCount,
			prometheus.GaugeValue,
			float64(viewerCount),
			twitchChannel.Name,
		)
	}

	if e.collectUserMetrics() {
		e.Logger.Info("User Chanel Metrics not yet available, only retrieving basic metrics")
	}
}

// Returns 1 if broadcasting, 0 if not
func (e *Exporter) IsLive(channelName string) (isLive int, err error) {
	foundChannels, err := e.client.SearchChannels(&helix.SearchChannelsParams{
		Channel: channelName,
	})
	if err != nil {
		return 0, err
	}

	if foundChannels.StatusCode != 200 {
		return 0, fmt.Errorf(foundChannels.Error)
	}

	for _, channel := range foundChannels.Data.Channels {
		if strings.EqualFold(channel.DisplayName, channelName) && channel.IsLive {
			return 1, nil
		}
	}

	return 0, nil
}

func (e *Exporter) ViewerCount(channelName string) (count int, err error) {
	stream, err := e.client.GetStreams(&helix.StreamsParams{
		UserLogins: []string{channelName},
	})
	if err != nil {
		return 0, err
	}

	if stream.StatusCode != 200 {
		return 0, fmt.Errorf(stream.Error)
	}

	if len(stream.Data.Streams) <= 0 {
		return 0, nil
	}

	return stream.Data.Streams[0].ViewerCount, nil
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
			prometheus.BuildFQName(namespace, "", "viewer_count"),
			"Channel current viewer count",
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
