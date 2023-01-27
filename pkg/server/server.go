package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	legacyV1 "github.com/go-semantic-release/plugin-registry/pkg/legacy/v1"
	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router             chi.Router
	log                *logrus.Logger
	db                 *firestore.Client
	ghClient           *github.Client
	ghMutex            sync.Mutex
	adminAccessToken   string
	rateLimitPerMinute int
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

func New(log *logrus.Logger, db *firestore.Client, ghClient *github.Client, adminAccessToken string) *Server {
	router := chi.NewRouter()
	server := &Server{
		router:             router,
		log:                log,
		db:                 db,
		ghClient:           ghClient,
		adminAccessToken:   adminAccessToken,
		rateLimitPerMinute: 1,
	}
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(server.logMiddleware)
	router.Use(server.recoverMiddleware)

	router.Use(middleware.Timeout(60 * time.Second))

	router.NotFound(server.notFoundHandler)
	router.MethodNotAllowed(server.methodNotAllowedHandler)

	// serve legacy API
	router.Handle("/api/v1/*", http.StripPrefix("/api/v1/", http.FileServer(http.FS(legacyV1.PluginIndex))))

	router.Route("/api/v2/plugins", func(r chi.Router) {
		r.Get("/", server.listPlugins)
		r.Get("/{plugin}", server.getPlugin)
		r.Get("/{plugin}/versions/{version}", server.getPlugin)

		// routes to update the plugin index
		r.With(server.authMiddleware).Group(func(r chi.Router) {
			r.Put("/", server.updateAllPlugins)
			r.Put("/{plugin}", server.updatePlugin)
			r.Put("/{plugin}/versions/{version}", server.updatePlugin)
		})
	})

	return server
}
