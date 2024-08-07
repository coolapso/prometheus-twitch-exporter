# PROMETHEUS TWITCH EXPORTER

Prometheus twitch exporter to monitor twitch streams with prometheus. With support Oauth support, for both user and application tokens. 

> [!IMPORTANT]
> This project is still under developemt, breaking changes are to be expected, and some features may not yet be available
> Even tho user tokens are already available and functional, User specific metrics are not yet available

## Install

Currently you can only grab one of the binaries provided in the releases page, or run it using docker. Check each use case examples for more details.

## Exported Metrics

| Metric | Meaning | Labels | type |
| ------ | ------- | ------ | ---- |
| twitch_is_live | If twitch channel is broadcasting | name | gauge |
| twitch_viewer_count | Channel current viewer count | name | gauge |
| twitch_channel_followers | The number of channel followers | name | gauge |
| twitch_channel_subscribers | The number of channel subscribers | name, tier, gifted | gauge |

## Usage

```
Usage:
  twitch-exporter [flags]

Flags:
      --address string            The address to access the exporter used for OAuth redirect URI (default "localhost")
      --client.id string          Twitch client ID
      --client.secret string      Twitch client secret
  -h, --help                      Help for twitch-exporter
      --listen.port string        Port to listen on (default "9184")
      --log.format string         Exporter log format, text or JSON (default "text")
      --log.level string          Exporter log level (default "info")
      --metrics.path string       Path to expose metrics at (default "/metrics")
      --refresh.token string      Twitch refresh token
      --twitch.channels strings   List of channels to get basic metrics from
      --twitch.user string        The user associated with the user token to get extra metrics from
      --user.token                Use the provided token as a user token
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

# Contributions

Improvements and suggestions are always welcome, feel free to check for any open issues, open a new Issue or Pull Request

If you like this project and want to support / contribute in a different way you can always: 

<a href="https://www.buymeacoffee.com/coolapso" target="_blank">
  <img src="https://cdn.buymeacoffee.com/buttons/default-yellow.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" />
</a>

# Related projects 

* [twitch_exporter](https://github.com/damoun/twitch_exporter)

