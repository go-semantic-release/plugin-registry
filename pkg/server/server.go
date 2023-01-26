package server

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/go-github/v43/github"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

type Server struct {
	router   *httprouter.Router
	log      *logrus.Logger
	db       *firestore.Client
	ghClient *github.Client
	ghMutex  sync.Mutex
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.log.Printf("%s %s", request.Method, request.URL.EscapedPath())
	if request.Method != http.MethodOptions {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	s.router.ServeHTTP(writer, request)
}

func (s *Server) writeJSONError(w io.Writer, e string) {
	s.writeJSON(w, map[string]string{"error": e})
}

func (s *Server) writeJSON(w io.Writer, d any) {
	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		s.log.Error(err)
	}
}

func (s *Server) write500Error(w http.ResponseWriter, err error, msg string) {
	s.log.Error(err)
	w.WriteHeader(500)
	s.writeJSONError(w, msg)
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	s.writeJSONError(w, "not found")
}

func (s *Server) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	s.writeJSONError(w, "method now allowed")
}

func (s *Server) globalOptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Access-Control-Request-Method") != "" {
		w.Header().Set("Access-Control-Allow-Methods", w.Header().Get("Allow"))
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rateLimitHandler(h httprouter.Handle, maxPerMinute int) httprouter.Handle {
	rl := ratelimit.New(maxPerMinute, ratelimit.Per(time.Minute), ratelimit.WithoutSlack)
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		rl.Take()
		if ctxErr := r.Context().Err(); ctxErr != nil {
			s.write500Error(w, ctxErr, "internal server error")
			return
		}
		h(w, r, ps)
	}
}

func New(log *logrus.Logger, db *firestore.Client, ghClient *github.Client) *Server {
	server := &Server{
		router:   httprouter.New(),
		log:      log,
		db:       db,
		ghClient: ghClient,
	}
	server.router.NotFound = http.HandlerFunc(server.notFoundHandler)
	server.router.MethodNotAllowed = http.HandlerFunc(server.methodNotAllowedHandler)
	server.router.GlobalOPTIONS = http.HandlerFunc(server.globalOptionsHandler)

	server.router.GET("/api/v2/plugins", server.listPlugins)
	// TODO: only enable this endpoint for authenticated users
	server.router.PUT("/api/v2/plugins", server.rateLimitHandler(server.updateAllPlugins, 1))

	server.router.GET("/api/v2/plugins/:plugin", server.getPlugin)
	server.router.PUT("/api/v2/plugins/:plugin", server.rateLimitHandler(server.updatePlugin, 1))

	server.router.GET("/api/v2/plugins/:plugin/:version", server.getPlugin)
	server.router.PUT("/api/v2/plugins/:plugin/:version", server.rateLimitHandler(server.updatePlugin, 1))
	return server
}
