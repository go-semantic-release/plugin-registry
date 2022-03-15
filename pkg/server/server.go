package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router *httprouter.Router
	log    *logrus.Logger
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

func (s *Server) writeJSON(w io.Writer, d interface{}) {
	err := json.NewEncoder(w).Encode(d)
	if err != nil {
		s.log.Error(err)
	}
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

func New(log *logrus.Logger) *Server {
	server := &Server{
		router: httprouter.New(),
		log:    log,
	}
	server.router.NotFound = http.HandlerFunc(server.notFoundHandler)
	server.router.MethodNotAllowed = http.HandlerFunc(server.methodNotAllowedHandler)
	server.router.GlobalOPTIONS = http.HandlerFunc(server.globalOptionsHandler)

	server.router.GET("/api/v2/plugins", server.listPlugins)
	//server.router.GET("/api/v2/plugins/:plugin", Index)
	//server.router.GET("/api/v2/plugins/:plugin/:version", Index)
	return server
}
