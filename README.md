# PROMETHEUS TWITCH EXPORTER

[![release](https://github.com/coolapso/prometheus-twitch-exporter/actions/workflows/release.yaml/badge.svg)](https://github.com/coolapso/prometheus-twitch-exporter/actions/workflows/release.yaml)
![GitHub Tag](https://img.shields.io/github/v/tag/coolapso/prometheus-twitch-exporter?logo=semver&label=semver&labelColor=gray&color=green)
[![Docker image version](https://img.shields.io/docker/v/coolapso/twitch-exporter/latest?logo=docker)](https://hub.docker.com/r/coolapso/twitch-exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/coolapso/prometheus-twitch-exporter)](https://goreportcard.com/report/github.com/coolapso/prometheus-twitch-exporter)
![GitHub Sponsors](https://img.shields.io/github/sponsors/coolapso?style=flat&logo=githubsponsors)

Prometheus twitch exporter to monitor twitch streams with prometheus. With support Oauth support, for both user and application tokens. 

## Install

Currently you can only grab one of the binaries provided in the releases page, or run it using docker. Check each use case examples for more details.

## Exported Metrics

| Metric | Meaning | Labels | type |
| ------ | ------- | ------ | ---- |
| twitch_is_live | If twitch channel is broadcasting | name | gauge |
| twitch_viewer_total | Channel current viewer count | name | gauge |
| twitch_channel_followers_total | The number of channel followers | name | gauge |
| twitch_channel_subscribers_total | The number of channel subscribers | name | gauge |

## Usage

```
Usage:
  twitch-exporter [flags]

Flags:
      --access.token string       twitch user access token
      --address string            The address to access the exporter used for oauth redirect uri (default "localhost")
      --client.id string          twitch client id
      --client.secret string      twitch client secret
  -h, --help                      help for twitch-exporter
      --listen.port string        Port to listen at (default "9184")
      --log.format string         Exporter log format, text or json (default "text")
      --log.level string          Exporter log level (default "info")
      --metrics.path string       Path to expose metrics at (default "/metrics")
      --refresh.token string      twitch refresh token
      --twitch.channels strings   List of channels to get basic metrics from
      --twitch.user string        The user associated with the user token to get extra metrics from
      --user.token                If going to use the provided token as a user token
```

You can also use environment variables. The most accurate list for them is available [here](cmd/root.go).

Twitch provides two types of authentication methods:

### Using Application Tokens

When using an application token, you are only granted access to some generic public endpoints. This exporter can only gather a basic set of metrics from any channel available on Twitch.

1. Go to the [Twitch Developer Portal](https://dev.twitch.tv/), log in, and register a new application.
2. The OAuth Redirect URL doesn't matter, but you can add the address to twitch exporter `http://<exporterAddress>:9184`.
3. Manage your application, copy the client ID, and generate a new secret.
4. Start the exporter and provide the `--client.id <ClientID>`, `--client.secret <ClientSecret>`, and the `--twitch.channels <ChannelsList>` you wish to monitor.
5. Access the exporter at `http://<exporterAddress>:9184/metrics`.

#### Examples

Using flags:
```
./twitch-exporter --client.id <ClientID> --client.secret <ClientSecret> --twitch.channels "chan1,chan2,chan3"
```

Using environment variables:
```
TWITCH_CLIENT_ID="<ClientID>" TWITCH_CLIENT_SECRET="<ClientSecret>" TWITCH_CHANNELS="chan1 chan2 chan3" ./twitch-exporter
```

With Docker, using flags:
```
docker run -d -p 9184:9184 \
        coolapso/twitch-exporter \
        --client.id=<ClientID> \
        --client.secret=<ClientSecret> \
        --twitch.channels="chan1,chan2,chan3"
```

With Docker, using environment variables:
```
docker run -d -p 9184:9184 \
        -e TWITCH_CLIENT_ID=<ClientID> \
        -e TWITCH_CLIENT_SECRET=<ClientSecret> \
        -e TWITCH_CHANNELS="chan1 chan2 chan3" \
        coolapso/twitch-exporter
```

### Using User Tokens

When using user tokens, you are granted access to all features and metrics available for application tokens, plus information specific to your own channel/username.

1. Go to the [Twitch Developer Portal](https://dev.twitch.tv/), log in, and register a new application.
2. Fill in the OAuth Redirect URL correctly, as it is used to redirect you to the Prometheus Twitch exporter to finish the user authentication flow. You should use the address where the exporter will be reachable (from your browser).
    * http://localhost:9184
    * http://<MachineIP>:9184
    * http://twitchexporter.mydomain.com:9184
3. Manage your application, copy the client ID, and generate a new secret.
4. Start the exporter and provide the `--client.id <ClientID>`, `--client.secret <ClientSecret>`, set the `--user.token` flag, and provide the username associated with the OAuth token with the `--twitch.user <UserName>`.
5. Grab the authentication URL from the logs or from `http://<ExporterAddress>:9184/` and complete the authentication flow.
6. You can also monitor other Twitch channels at the same time; however, you can only get basic metrics for those channels.

#### Examples

Using flags:
```
./twitch-exporter --client.id <ClientID> --client.secret <ClientSecret> --twitch.channels "chan1,chan2,chan3" --user.token --twitch.user cool4pso 
```

Using environment variables:
```
TWITCH_CLIENT_ID="<ClientID>" TWITCH_CLIENT_SECRET="<ClientSecret>" TWITCH_USER_TOKEN=true TWITCH_USER=cool4pso TWITCH_CHANNELS="chan1 chan2 chan3" ./twitch-exporter
```

With Docker, using flags:
```
docker run -d -p 9184:9184 \
        coolapso/twitch-exporter \
        --address="192.168.1.5"
        --client.id=<ClientID> \
        --client.secret=<ClientSecret> \
        --twitch.channels="chan1,chan2,chan3" \
        --user.token \
        --twitch.user cool4pso
```

With Docker, using environment variables:
```
docker run -d -p 9184:9184 \
        -e ADDRESS="192.168.1.5"
        -e TWITCH_CLIENT_ID=<ClientID> \
        -e TWITCH_CLIENT_SECRET=<ClientSecret> \
        -e TWITCH_CHANNELS="chan1 chan2 chan3" \
        -e TWITCH_USER_TOKEN=true \
        coolapso/twitch-exporter
```

#### Pre-generated access token

You can also pre-generate the access token and refresh token, for example with Twitch CLI:

```
twitch token get -u --scopes "channel:read:subscriptions"
```

Then provide them to the application using the flags or corresponding environment variables. This way, you won't have to handle the authentication flow every time.

# Contributions

Improvements and suggestions are always welcome, feel free to check for any open issues, open a new Issue or Pull Request

If you like this project and want to support / contribute in a different way you can always [:heart: Sponsor Me](https://github.com/sponsors/coolapso) or

<a href="https://www.buymeacoffee.com/coolapso" target="_blank">
  <img src="https://cdn.buymeacoffee.com/buttons/default-yellow.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" />
</a>

# Related projects 

* [twitch_exporter](https://github.com/damoun/twitch_exporter)

