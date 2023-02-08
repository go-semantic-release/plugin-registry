package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-semantic-release/plugin-registry/internal/metrics"
	"github.com/patrickmn/go-cache"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type (
	cacheKeyPrefix string
	cacheKey       string
)

const (
	cacheKeyPrefixBatchRequest cacheKeyPrefix = "batch"
	cacheKeyPrefixRequest      cacheKeyPrefix = "request"
	cacheKeyPrefixGitHub       cacheKeyPrefix = "github"
)

func (s *Server) getCacheKeyFromRequest(r *http.Request) cacheKey {
	return cacheKey(fmt.Sprintf("%s/%s:%s", cacheKeyPrefixRequest, r.Method, r.URL.EscapedPath()))
}

func (s *Server) getCacheKeyPrefixFromPluginName(pluginName string) cacheKey {
	return cacheKey(fmt.Sprintf("%s/%s:/api/v2/plugins/%s", cacheKeyPrefixRequest, http.MethodGet, pluginName))
}

func (s *Server) getCacheKeyWithPrefix(p cacheKeyPrefix, key string) cacheKey {
	return cacheKey(fmt.Sprintf("%s/%s", p, key))
}

func (s *Server) getFromCache(ctx context.Context, k cacheKey) (any, bool) {
	strKey := string(k)
	val, ok := s.cache.Get(strKey)
	if ok {
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.TagCacheKey, strKey))
		stats.Record(ctx, metrics.CounterCacheHit.M(1))
	}
	return val, ok
}

func (s *Server) setInCache(ctx context.Context, k cacheKey, v any, expiration ...time.Duration) {
	strKey := string(k)
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TagCacheKey, strKey))
	stats.Record(ctx, metrics.CounterCacheMiss.M(1))
	exp := cache.DefaultExpiration
	if len(expiration) > 0 {
		exp = expiration[0]
	}
	s.cache.Set(strKey, v, exp)
}

func (s *Server) invalidateByPrefix(prefix cacheKey) {
	for k := range s.cache.Items() {
		if strings.HasPrefix(k, string(prefix)) {
			s.cache.Delete(k)
		}
	}
}

func (s *Server) cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.DisableRequestCache {
			next.ServeHTTP(w, r)
			return
		}
		if k, ok := s.getFromCache(r.Context(), s.getCacheKeyFromRequest(r)); ok {
			w.Header().Set("X-Go-Cache", "HIT")
			s.writeJSON(w, k)
			return
		}
		next.ServeHTTP(w, r)
	})
}
