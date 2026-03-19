package server

import (
	"net/http"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/learnings"
)

func (s *Server) handleLearnings(w http.ResponseWriter, r *http.Request) {
	topic := r.PathValue("topic")
	if topic == "" {
		http.Error(w, "topic required", http.StatusBadRequest)
		return
	}

	st := learnings.New(config.LearningsDir())
	content, err := st.Read(topic)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(content))
}
