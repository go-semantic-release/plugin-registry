package server

import (
	"net/http"
	"sync"

	"cloud.google.com/go/firestore"
	legacyV1 "github.com/go-semantic-release/plugin-registry/pkg/legacy/v1"
	"github.com/google/go-github/v50/github"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router             *httprouter.Router
	log                *logrus.Logger
	db                 *firestore.Client
	ghClient           *github.Client
	ghMutex            sync.Mutex
	adminAccessToken   string
	rateLimitPerMinute int
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.log.Printf("%s %s", request.Method, request.URL.EscapedPath())
	if request.Method != http.MethodOptions {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	s.router.ServeHTTP(writer, request)
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSONError(w, http.StatusNotFound, nil, "not found")
}

func (s *Server) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSONError(w, http.StatusMethodNotAllowed, nil, "method now allowed")
}

func (s *Server) globalOptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Access-Control-Request-Method") != "" {
		w.Header().Set("Access-Control-Allow-Methods", w.Header().Get("Allow"))
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
	}
	w.WriteHeader(http.StatusNoContent)
}

func New(log *logrus.Logger, db *firestore.Client, ghClient *github.Client, adminAccessToken string) *Server {
	server := &Server{
		router:             httprouter.New(),
		log:                log,
		db:                 db,
		ghClient:           ghClient,
		adminAccessToken:   adminAccessToken,
		rateLimitPerMinute: 1,
	}
	server.router.NotFound = http.HandlerFunc(server.notFoundHandler)
	server.router.MethodNotAllowed = http.HandlerFunc(server.methodNotAllowedHandler)
	server.router.GlobalOPTIONS = http.HandlerFunc(server.globalOptionsHandler)

	// serve legacy API
	server.router.ServeFiles("/api/v1/*filepath", http.FS(legacyV1.PluginIndex))

	server.router.GET("/api/v2/plugins", server.listPlugins)
	server.router.GET("/api/v2/plugins/:plugin", server.getPlugin)
	server.router.GET("/api/v2/plugins/:plugin/:version", server.getPlugin)

	// routes to update the plugin index
	server.router.PUT("/api/v2/plugins", server.authMiddleware(server.updateAllPlugins))
	server.router.PUT("/api/v2/plugins/:plugin/:version", server.authMiddleware(server.updatePlugin))
	server.router.PUT("/api/v2/plugins/:plugin", server.authMiddleware(server.updatePlugin))

	return server
}
