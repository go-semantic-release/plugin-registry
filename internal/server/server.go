package server

import (
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-semantic-release/plugin-registry/internal/config"
	legacyV1 "github.com/go-semantic-release/plugin-registry/internal/legacy/v1"
	"github.com/google/go-github/v50/github"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

type Server struct {
	router   chi.Router
	log      *logrus.Logger
	db       *firestore.Client
	ghClient *github.Client
	storage  *s3.Client
	config   *config.ServerConfig
	cache    *cache.Cache

	ghSemaphore           *semaphore.Weighted
	batchArchiveSemaphore *semaphore.Weighted
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSONError(w, r, http.StatusNotFound, fmt.Errorf("not found"))
}

func (s *Server) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSONError(w, r, http.StatusMethodNotAllowed, fmt.Errorf("method now allowed"))
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{
		"service": "go-semantic-release plugin registry",
		"stage":   s.config.Stage,
		"version": s.config.Version,
	})
}

func (s *Server) apiV2Routes(r chi.Router) {
	r.Route("/plugins", func(r chi.Router) {
		r.With(s.cacheMiddleware).Group(func(r chi.Router) {
			r.Get("/", s.listPlugins)
			r.Get("/{plugin}", s.getPlugin)
			r.Get("/{plugin}/versions", s.listPluginVersions)
			r.Get("/{plugin}/versions/{version}", s.getPlugin)
		})

		r.Post("/_batch", s.batchGetPlugins)

		// routes to update the plugin index
		r.With(s.authMiddleware).Group(func(r chi.Router) {
			r.Put("/", s.updateAllPlugins)
			r.Put("/{plugin}", s.updatePlugin)
			r.Put("/{plugin}/versions/{version}", s.updatePlugin)
		})
	})
}

func New(log *logrus.Logger, db *firestore.Client, ghClient *github.Client, storage *s3.Client, serverCfg *config.ServerConfig) *Server {
	router := chi.NewRouter()
	server := &Server{
		router:                router,
		log:                   log,
		db:                    db,
		ghClient:              ghClient,
		storage:               storage,
		config:                serverCfg,
		cache:                 cache.New(15*time.Minute, 30*time.Minute),
		ghSemaphore:           semaphore.NewWeighted(1),
		batchArchiveSemaphore: semaphore.NewWeighted(1),
	}
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(server.logMiddleware)
	router.Use(server.recoverMiddleware)
	// router.Use(middleware.Recoverer)

	router.Use(middleware.Timeout(5 * time.Minute))

	router.NotFound(server.notFoundHandler)
	router.MethodNotAllowed(server.methodNotAllowedHandler)

	router.Get("/", server.indexHandler)

	// serve legacy API
	router.Handle("/api/v1/*", http.StripPrefix("/api/v1/", http.FileServer(http.FS(legacyV1.PluginIndex))))

	router.Route("/api/v2", server.apiV2Routes)

	// downloads route
	router.Get("/downloads/{os}/{arch}/semantic-release", server.downloadLatestSemRelBinary)

	return server
}
