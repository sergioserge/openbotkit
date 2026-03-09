package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	wasrc "github.com/priyanshujain/openbotkit/source/whatsapp"
	"github.com/priyanshujain/openbotkit/store"
)

type whatsappSendRequest struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type whatsappSendResponse struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleWhatsAppSend(w http.ResponseWriter, r *http.Request) {
	var req whatsappSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.To == "" {
		writeError(w, http.StatusBadRequest, "to field is required")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text field is required")
		return
	}

	client, err := wasrc.NewClient(r.Context(), s.cfg.WhatsAppSessionDBPath())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("create client: %v", err))
		return
	}
	defer client.Disconnect()

	if !client.IsAuthenticated() {
		writeError(w, http.StatusPreconditionFailed, "WhatsApp not authenticated")
		return
	}

	db, err := store.Open(store.Config{
		Driver: s.cfg.WhatsApp.Storage.Driver,
		DSN:    s.cfg.WhatsAppDataDSN(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("open db: %v", err))
		return
	}
	defer db.Close()

	if err := wasrc.Migrate(db); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("migrate: %v", err))
		return
	}

	result, err := wasrc.SendText(r.Context(), client, db, wasrc.SendInput{
		ChatJID: req.To,
		Text:    req.Text,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("send failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(whatsappSendResponse{
		MessageID: result.MessageID,
		Timestamp: result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	})
}
