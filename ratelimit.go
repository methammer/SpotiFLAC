package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	rlMaxAttempts  = 10              // tentatives autorisées par fenêtre
	rlWindow       = time.Minute     // durée de la fenêtre
	rlBlockDur     = 5 * time.Minute // durée du blocage après dépassement
	rlCleanupEvery = 10 * time.Minute
)

type rlEntry struct {
	attempts  int
	windowEnd time.Time
	blockedUntil time.Time
}

// LoginRateLimiter limite les tentatives de login par IP.
type LoginRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rlEntry
}

func NewLoginRateLimiter() *LoginRateLimiter {
	rl := &LoginRateLimiter{entries: make(map[string]*rlEntry)}
	go rl.cleanupLoop()
	return rl
}

// Allow retourne true si la requête est autorisée, false si elle doit être rejetée.
func (rl *LoginRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[ip]
	if !ok {
		e = &rlEntry{}
		rl.entries[ip] = e
	}

	// IP bloquée ?
	if now.Before(e.blockedUntil) {
		return false
	}

	// Fenêtre expirée → remettre à zéro
	if now.After(e.windowEnd) {
		e.attempts = 0
		e.windowEnd = now.Add(rlWindow)
	}

	e.attempts++
	if e.attempts > rlMaxAttempts {
		e.blockedUntil = now.Add(rlBlockDur)
		return false
	}
	return true
}

func (rl *LoginRateLimiter) cleanupLoop() {
	t := time.NewTicker(rlCleanupEvery)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, e := range rl.entries {
			if now.After(e.blockedUntil) && now.After(e.windowEnd) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// remoteIP extrait l'IP de la requête (tient compte de X-Forwarded-For
// uniquement si la connexion directe vient d'une IP privée / loopback,
// i.e. derrière un reverse proxy de confiance).
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	// Derrière un reverse proxy de confiance → utiliser X-Forwarded-For
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Prendre la première IP (la plus à gauche = le vrai client)
			for _, part := range splitComma(xff) {
				if candidate := net.ParseIP(trimSpace(part)); candidate != nil {
					return candidate.String()
				}
			}
		}
	}
	return host
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
