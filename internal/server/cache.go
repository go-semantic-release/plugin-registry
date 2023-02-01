package server

import (
	"fmt"
	"net/http"

	"github.com/patrickmn/go-cache"
)

type (
	cacheKeyPrefix string
	cacheKey       string
)

const (
	cacheKeyPrefixBatchRequest cacheKeyPrefix = "batch"
	cacheKeyPrefixRequest      cacheKeyPrefix = "request"
)

func (s *Server) getCacheKeyFromRequest(r *http.Request) cacheKey {
	return cacheKey(fmt.Sprintf("%s/%s:%s", cacheKeyPrefixRequest, r.Method, r.URL.EscapedPath()))
}

func (s *Server) getCacheKeyWithPrefix(p cacheKeyPrefix, key string) cacheKey {
	return cacheKey(fmt.Sprintf("%s/%s", p, key))
}

func (s *Server) getFromCache(k cacheKey) (any, bool) {
	return s.cache.Get(string(k))
}

func (s *Server) setInCache(k cacheKey, v any) {
	s.cache.Set(string(k), v, cache.DefaultExpiration)
}
