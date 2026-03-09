package server

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
)

type dbRequest struct {
	SQL string `json:"sql"`
}

type dbResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func (s *Server) handleDB(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")

	var req dbRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql field is required")
		return
	}

	trimmed := strings.TrimSpace(strings.ToUpper(req.SQL))
	if !strings.HasPrefix(trimmed, "SELECT") {
		writeError(w, http.StatusBadRequest, "only SELECT queries are allowed")
		return
	}
	if strings.Contains(req.SQL, ";") {
		writeError(w, http.StatusBadRequest, "multiple statements are not allowed")
		return
	}

	dsn, err := s.cfg.SourceDataDSN(source)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown source")
		return
	}

	// Open in read-only mode to prevent any writes.
	db, err := sql.Open("sqlite", dsn+"?mode=ro")
	if err != nil {
		slog.Error("db handler: open database", "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to open database")
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), req.SQL)
	if err != nil {
		slog.Error("db handler: query", "source", source, "error", err)
		writeError(w, http.StatusBadRequest, "query failed")
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		slog.Error("db handler: columns", "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to read columns")
		return
	}

	var result [][]string
	for rows.Next() {
		vals := make([]sql.NullString, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			slog.Error("db handler: scan row", "source", source, "error", err)
			writeError(w, http.StatusInternalServerError, "failed to read row")
			return
		}
		row := make([]string, len(columns))
		for i, v := range vals {
			if v.Valid {
				row[i] = v.String
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		slog.Error("db handler: rows iteration", "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, "query error")
		return
	}

	if result == nil {
		result = [][]string{}
	}

	resp := dbResponse{Columns: columns, Rows: result}
	if resp.Columns == nil {
		resp.Columns = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
