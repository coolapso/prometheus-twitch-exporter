package httpServer

import (
	"github.com/coolapso/prometheus-twitch-exporter/internal/collectors"
	"github.com/prometheus/client_golang/prometheus"
	promCollectors "github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"html/template"
	"net/http"
)

const (
	rootTemplate string = `<html>
	 <head><title>Prometheus Twitch Exporter</title></head>
	 <body>
		 <h1>Prometheus Twitch Exporter</h1>
		 <p>Metrics at: <a href='{{ .MetricsPath }}'>{{ .MetricsPath }}</a></p>
		 {{ if .UserToken }}
		 <p>Authorize Prometheus twitch exporter <a href='{{ .ApiSettings.AuthorizationURL }}'>here</a></p>
		 {{ end }}
		 <p>Source: <a href='https://github.com/coolapso/prometheus-twitch-exporter'>github.com/coolapso/prometheus-twitch-exporter</a></p>
	 </body>
	 </html>`
)

// TODO: Re-eneable remaining collectors
func NewServer(e *collectors.Exporter) *http.Server {
	s := e.Settings
	logger := e.Logger
	t := template.Must(template.New("root").Parse(rootTemplate))

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		e,
		versioncollector.NewCollector("twitch_exporter"),
		promCollectors.NewBuildInfoCollector(),
		promCollectors.NewGoCollector(),
	)

	promHandlerOpts := promhttp.HandlerOpts{
		Registry: reg,
	}

	// Metrics handler
	http.Handle(s.MetricsPath, promhttp.HandlerFor(reg, promHandlerOpts))

	// Root Page handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, e.Settings)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		query := r.URL.Query()
		if len(query) > 0 {
			if len(query["code"]) > 0 {
				s.ApiSettings.AuthorizationCode = query["code"][0]
				logger.Info("Prometheus Twitch Exporter authorized by user")
				_, err := w.Write([]byte("exporter has been authorized"))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}


	})


	return &http.Server{Addr: ":" + s.ListenPort}
}
