package main

// ─────────────────────────────────────────────────────────────────────────────
// SSE — Server-Sent Events pour la progression des jobs
//
// GET /api/v1/jobs/stream → text/event-stream
//
// Événements :
//   event: job_update   — état d'un job (pending/downloading/done/failed/skipped)
//   event: connected    — snapshot initial de la queue au moment de la connexion
// ─────────────────────────────────────────────────────────────────────────────

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

type JobEvent struct {
	Type string `json:"type"`
	Job  *Job   `json:"job"`
}

// ─────────────────────────────────────────────────────────────────────────────
// SSEHub — fan-out des événements vers tous les clients connectés
// ─────────────────────────────────────────────────────────────────────────────

type SSEHub struct {
	mu   sync.RWMutex
	subs map[chan JobEvent]struct{}
}

func newSSEHub() *SSEHub {
	return &SSEHub{subs: make(map[chan JobEvent]struct{})}
}

// subscribe crée un canal dédié au client et l'enregistre dans le hub.
func (h *SSEHub) subscribe() chan JobEvent {
	ch := make(chan JobEvent, 32) // buffer pour absorber les pics
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// unsubscribe retire le canal du hub et le ferme.
func (h *SSEHub) unsubscribe(ch chan JobEvent) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

// publish diffuse un événement à tous les abonnés.
// Les canaux trop lents sont ignorés (select default) pour ne pas bloquer.
func (h *SSEHub) publish(event JobEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- event:
		default:
			// consommateur trop lent — on skip plutôt que bloquer
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers d'écriture SSE
// ─────────────────────────────────────────────────────────────────────────────

// sendSSEEvent écrit un événement SSE et flush immédiatement.
func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
	flusher.Flush()
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler SSE — GET /api/v1/jobs/stream
// ─────────────────────────────────────────────────────────────────────────────

func (s *Server) v1JobsStream(w http.ResponseWriter, r *http.Request) {
	// Vérifier que le client accepte les SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeV1Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // désactive le buffering nginx

	// Snapshot initial — envoyer tous les jobs existants
	if jobs, err := s.ctr.Jobs.GetAllJobs(); err == nil {
		for i := range jobs {
			sendSSEEvent(w, flusher, "job_update", &jobs[i])
		}
	}

	// Filtrer les jobs selon le user courant (API key ou JWT)
	user := GetUserFromContext(r)

	// S'abonner au hub
	ch := s.ctr.Jobs.hub.subscribe()
	defer s.ctr.Jobs.hub.unsubscribe(ch)

	// Signal de connexion établie
	sendSSEEvent(w, flusher, "connected", map[string]string{"status": "ok"})

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			// Filtrer par userID si non-admin
			if user != nil && !user.IsAdmin && event.Job != nil &&
				event.Job.UserID != "" && event.Job.UserID != user.UserID {
				continue
			}
			sendSSEEvent(w, flusher, event.Type, event.Job)
		case <-r.Context().Done():
			return
		}
	}
}
