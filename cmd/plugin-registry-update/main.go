package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-semantic-release/plugin-registry/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version = "dev"

var defaultPluginRegistryURLs = []string{
	"https://registry.go-semantic-release.xyz",
	"https://registry-staging.go-semantic-release.xyz",
}

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	cmd := &cobra.Command{
		Use:     "plugin-registry-update",
		Short:   "Trigger a plugin registry update",
		Version: version,
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(log, cmd, args); err != nil {
				log.Errorf("ERROR: %v", err)
				os.Exit(1)
			}
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.PersistentFlags().StringArrayP("registry-url", "r", defaultPluginRegistryURLs, "the plugin registry URL")
	cmd.PersistentFlags().String("admin-access-token", os.Getenv("PLUGIN_REGISTRY_ADMIN_ACCESS_TOKEN"), "admin access token")
	cmd.PersistentFlags().StringP("plugin-name", "p", "", "the plugin name")
	cmd.PersistentFlags().StringP("plugin-version", "v", "", "the plugin version")
	cmd.PersistentFlags().SortFlags = false

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func run(log *logrus.Logger, cmd *cobra.Command, _ []string) error {
	log.Infof("starting plugin-registry-update (version=%s)", version)
	registryURLs := must(cmd.PersistentFlags().GetStringArray("registry-url"))
	if len(registryURLs) == 0 {
		return errors.New("no registry URLs provided")
	}
	adminAccessToken := must(cmd.PersistentFlags().GetString("admin-access-token"))
	if adminAccessToken == "" {
		return errors.New("no admin access token provided")
	}
	pluginName := must(cmd.PersistentFlags().GetString("plugin-name"))
	pluginVersion := must(cmd.PersistentFlags().GetString("plugin-version"))

	fullUpdate := pluginName == "" && pluginVersion == ""
	if fullUpdate {
		log.Warn("triggering full registry update...")
	} else {
		log.Infof("triggering update for plugin: %s@%s.", pluginName, pluginVersion)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for _, url := range registryURLs {
		url = strings.TrimSuffix(url, "/")
		if !strings.HasSuffix(url, "/api/v2") {
			url += "/api/v2"
		}
		log.Infof("updating plugin registry: %s", url)
		c := client.New(url)
		err := c.UpdatePluginRelease(ctx, adminAccessToken, pluginName, pluginVersion)
		if err != nil {
			log.Errorf("failed to update plugin registry %s: %v", url, err)
		}
	}

	return nil
}
