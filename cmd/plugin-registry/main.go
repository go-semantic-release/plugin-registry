package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-semantic-release/plugin-registry/pkg/server"
	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func setupLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return log
}

func setupGitHubClient() (*github.Client, error) {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		return nil, fmt.Errorf("GITHUB_TOKEN is missing")
	}
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	return github.NewClient(oauthClient), nil
}

func getServerAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return os.Getenv("BIND_ADDRESS") + ":" + port
}

func run(log *logrus.Logger) error {
	log.Info("setting up GitHub client...")
	ghClient, err := setupGitHubClient()
	if err != nil {
		return err
	}

	log.Info("connecting to database...")
	db, err := firestore.NewClient(context.Background(), "go-semantic-release")
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    getServerAddr(),
		Handler: server.New(log, db, ghClient, os.Getenv("ADMIN_ACCESS_TOKEN")),
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

	log.Info("closing database...")
	if err := db.Close(); err != nil {
		log.Error(err)
	}

	log.Info("stopping server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); errors.Is(err, context.DeadlineExceeded) {
		log.Info("closing server...")
		if closeErr := srv.Close(); closeErr != nil {
			return closeErr
		}
	} else if err != nil {
		return err
	}
	log.Info("server stopped!")
	return nil
}

func main() {
	log := setupLogger()
	if err := run(log); err != nil {
		log.Fatal(err)
	}
}
