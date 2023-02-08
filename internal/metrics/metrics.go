package metrics

import (
	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	CounterSemRelDownloads = stats.Int64("semrel_downloads", "Number of semantic-release downloads", "1")
	CounterCacheHit        = stats.Int64("cache_hits", "Number of cache hits", "1")
	CounterCacheMiss       = stats.Int64("cache_misses", "Number of cache misses", "1")

	TagStage    = tag.MustNewKey("stage")
	TagOSArch   = tag.MustNewKey("os_arch")
	TagCacheKey = tag.MustNewKey("cache_key")
)

var views = []*view.View{
	{
		Name:        "semrel_downloads",
		Measure:     CounterSemRelDownloads,
		Description: "Number of semantic-release downloads",
		TagKeys:     []tag.Key{TagStage, TagOSArch},
		Aggregation: view.Count(),
	},
	{
		Name:        "cache_hits",
		Measure:     CounterCacheHit,
		Description: "Number of cache hits",
		TagKeys:     []tag.Key{TagStage, TagCacheKey},
		Aggregation: view.Count(),
	},
	{
		Name:        "cache_misses",
		Measure:     CounterCacheMiss,
		Description: "Number of cache misses",
		TagKeys:     []tag.Key{TagStage, TagCacheKey},
		Aggregation: view.Count(),
	},
}

func NewExporter(opt stackdriver.Options) (*stackdriver.Exporter, error) {
	err := view.Register(views...)
	if err != nil {
		return nil, err
	}
	exporter, err := stackdriver.NewExporter(opt)
	if err != nil {
		return nil, err
	}
	err = exporter.StartMetricsExporter()
	if err != nil {
		return nil, err
	}
	return exporter, nil
}
