package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	lgg "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var runningVMs map[string]RunningFirecracker = make(map[string]RunningFirecracker)
var ipByte byte = 3

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	lg := lgg.New()

	lg.SetFormatter(&lgg.JSONFormatter{})
	lg.SetOutput(os.Stdout)
	lg.SetLevel(lgg.DebugLevel)

	r := chi.NewMux()
	r.Use(middleware.Recoverer)
	r.Use(includeLogger(lg))
	r.Mount("/api", handler())

	lg.Infof("Listening on port 8080")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			cancel()
		case <-ctx.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(1)
	}()

	// for killing all running VMs
	defer func() {
		cleanup()
		cancel()
	}()

	g := errgroup.Group{}

	g.Go(func() error {
		return http.ListenAndServe(":8080", r)
	})

	<-ctx.Done()

	lg.Infoln("shutting down app")

	if err := g.Wait(); err != nil {
		lg.Fatal("main: runtime program terminated")
	}
}

// include context with logger in http server for downstream use
func includeLogger(lg *lgg.Logger) Middleware {

	return func(next http.Handler) http.Handler {

		f := func(w http.ResponseWriter, r *http.Request) {

			r = r.WithContext(ctxSetLogger(r.Context(), lg))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(f)
	}
}
