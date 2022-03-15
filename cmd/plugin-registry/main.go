package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-semantic-release/plugin-registry/pkg/server"
	"github.com/sirupsen/logrus"
)

func setupLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return log
}

func run(log *logrus.Logger) error {
	log.Println("starting server...")
	srv := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: server.New(log),
	}
	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Error(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	log.Println("stopping server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err == context.DeadlineExceeded {
		log.Println("closing server...")
		if err := srv.Close(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	log.Println("server stopped!")
	return nil
}

func main() {
	log := setupLogger()
	if err := run(log); err != nil {
		log.Fatal(err)
	}
}
