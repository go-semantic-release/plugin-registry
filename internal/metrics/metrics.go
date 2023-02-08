package metrics

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/go-semantic-release/plugin-registry/internal/config"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	CounterSemRelDownloads = stats.Int64("semrel_downloads", "Number of semantic-release downloads", "1")
	CounterCacheHit        = stats.Int64("cache_hits", "Number of cache hits", "1")
	CounterCacheMiss       = stats.Int64("cache_misses", "Number of cache misses", "1")

	TagOSArch = tag.MustNewKey("os_arch")
)

var views = []*view.View{
	{
		Name:        "semrel_downloads",
		Measure:     CounterSemRelDownloads,
		Description: "Number of semantic-release downloads",
		TagKeys:     []tag.Key{TagOSArch},
		Aggregation: view.Count(),
	},
	{
		Name:        "cache_hits",
		Measure:     CounterCacheHit,
		Description: "Number of cache hits",
		Aggregation: view.Count(),
	},
	{
		Name:        "cache_misses",
		Measure:     CounterCacheMiss,
		Description: "Number of cache misses",
		Aggregation: view.Count(),
	},
}

func NewExporter(cfg *config.ServerConfig) (*stackdriver.Exporter, error) {
	err := view.Register(views...)
	if err != nil {
		return nil, err
	}
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:    cfg.ProjectID,
		MetricPrefix: fmt.Sprintf("plugin-registry/%s", cfg.Stage),
	})
	if err != nil {
		return nil, err
	}
	err = exporter.StartMetricsExporter()
	if err != nil {
		return nil, err
	}
	return exporter, nil
}
