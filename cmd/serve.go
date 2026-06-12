package cmd

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/1zun4/zte-cpe-go/pkg/exporter"
	"github.com/1zun4/zte-cpe-go/pkg/g5ts"
	"github.com/1zun4/zte-cpe-go/pkg/mf289f"
	"github.com/1zun4/zte-cpe-go/pkg/router"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var (
	serveListen   string
	serveInterval int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Prometheus metrics server",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientFn := func() (router.RouterClient, error) {
			switch routerType {
			case "mf289f":
				return mf289f.New(routerURL)
			case "g5ts":
				return g5ts.New(routerURL)
			default:
				return nil, fmt.Errorf("unsupported router type: %s (use 'mf289f' or 'g5ts')", routerType)
			}
		}

		exp := exporter.NewExporter(clientFn, routerType, password)
		prometheus.MustRegister(exp)

		interval := time.Duration(serveInterval) * time.Second
		exp.Start(interval)

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "zte-cpe-go exporter running, model=%s, target=%s\n", routerType, routerURL)
		})

		log.Printf("Starting zte-cpe-go exporter on %s (model=%s, target=%s, interval=%ds)", serveListen, routerType, routerURL, serveInterval)
		return http.ListenAndServe(serveListen, mux)
	},
}

func init() {
	serveCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	serveCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	serveCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	serveCmd.Flags().StringVarP(&serveListen, "listen", "l", ":9101", "Listen address for the metrics server")
	serveCmd.Flags().IntVarP(&serveInterval, "interval", "i", 30, "Scrape interval in seconds")
	serveCmd.MarkFlagRequired("type")
	serveCmd.MarkFlagRequired("url")
	serveCmd.MarkFlagRequired("password")
}
