package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/store"
)

type memoryAddRequest struct {
	Content  string `json:"content"`
	Category string `json:"category"`
	Source   string `json:"source"`
}

type memoryAddResponse struct {
	ID int64 `json:"id"`
}

type memoryItem struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) openMemoryDB() (*store.DB, error) {
	dsn := s.cfg.UserMemoryDataDSN()
	db, err := store.Open(store.Config{
		Driver: s.cfg.UserMemory.Storage.Driver,
		DSN:    dsn,
	})
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	if err := memory.Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate memory db: %w", err)
	}
	return db, nil
}

func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	db, err := s.openMemoryDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	category := r.URL.Query().Get("category")

	var memories []memory.Memory
	if category != "" {
		memories, err = memory.ListByCategory(db, memory.Category(category))
	} else {
		memories, err = memory.List(db)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("list memories: %v", err))
		return
	}

	items := make([]memoryItem, len(memories))
	for i, m := range memories {
		items[i] = memoryItem{
			ID:        m.ID,
			Content:   m.Content,
			Category:  string(m.Category),
			Source:    m.Source,
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleMemoryAdd(w http.ResponseWriter, r *http.Request) {
	var req memoryAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Category == "" {
		req.Category = "preference"
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	db, err := s.openMemoryDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	id, err := memory.Add(db, req.Content, memory.Category(req.Category), req.Source, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("add memory: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(memoryAddResponse{ID: id})
}

func (s *Server) handleMemoryDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid memory ID")
		return
	}

	db, err := s.openMemoryDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	if err := memory.Delete(db, id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("delete memory: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
