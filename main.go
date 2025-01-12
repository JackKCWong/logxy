package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/JackKCWong/logxy/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "logxy",
	Short: "logxy is a http proxy for logging",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt16("port")
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			log.Fatal().Str("target", target).Msg("empty target url")
		}

		sslPEM, _ := cmd.Flags().GetString("ssl")

		targetURL, err := url.Parse(target)
		if err != nil {
			log.Fatal().Str("target", target).Err(err).Msg("")
		}

		server := &http.Server{
			Addr: ":" + strconv.Itoa(int(port)),
			Handler: &internal.SingleHostProxy{
				Target: targetURL,
				Client: &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				},
			},
		}

		go func() {
			log.Info().Int16("port", port).Msg("starting server")
			if sslPEM == "" {
				if err := server.ListenAndServe(); err != nil {
					log.Info().Msg(err.Error())
				}
			} else {
				if err := server.ListenAndServeTLS(sslPEM, sslPEM); err != nil {
					log.Info().Msg(err.Error())
				}
			}
		}()

		// Setting up signal capturing
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		// Waiting for SIGINT (kill -2)
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdown server")
		}
	},
}

func init() {
	cmd.Flags().Int16P("port", "p", 8080, "port to listen on")
	cmd.Flags().StringP("target", "t", "", "target url to proxy to")
	cmd.Flags().StringP("ssl", "", "", "start as https server using the specified pem file that contains both the cert and key")
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	if err := cmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to start logxy")
	}
}
