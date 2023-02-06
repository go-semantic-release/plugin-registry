package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-semantic-release/plugin-registry/internal/config"
	"github.com/go-semantic-release/plugin-registry/internal/plugin"
	"github.com/go-semantic-release/plugin-registry/internal/server"
	"github.com/sirupsen/logrus"
)

var version = "dev"

func setupLogger() *logrus.Logger {
	log := logrus.New()
	if os.Getenv("DEBUG") == "true" {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	} else {
		log.SetFormatter(&logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyLevel: "severity",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyTime:  "timestamp",
			},
		})
	}

	return log
}

func run(log *logrus.Logger) error {
	log.Infof("starting plugin-registry (version=%s)", version)
	log.Info("reading configuration...")
	cfg, err := config.NewServerConfigFromEnv()
	if err != nil {
		return err
	}

	// set global version
	cfg.Version = version

	if cfg.DisableRequestCache {
		log.Warn("request cache disabled")
	}

	log.Infof("connecting to database (stage=%s)...", cfg.Stage)
	// set global collection prefix
	plugin.CollectionPrefix = cfg.Stage

	db, err := firestore.NewClient(context.Background(), cfg.ProjectID)
	if err != nil {
		return err
	}

	log.Info("setting up S3 client...")
	s3Client, err := cfg.CreateS3Client()
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    cfg.GetServerAddr(),
		Handler: server.New(log, db, cfg.CreateGitHubClient(), s3Client, cfg),
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
