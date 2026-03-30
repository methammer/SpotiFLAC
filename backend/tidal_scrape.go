package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	// Fallback web player client_id (peut être périmé — utilisé uniquement si le scraping
	// ET l'Android fallback échouent, ce qui ne devrait jamais arriver en pratique).
	tidalWebClientIDFallback = "CzET4vdadNUFQ5JU"

	// Credentials Android — stables entre les déploiements du web player.
	// Utilisés comme fallback si le scraping du web player échoue.
	tidalAndroidClientID    = "6BDSRdpK9hqEBTgU"
	tidalAndroidRedirectURI = "https://tidal.com/android/login/auth"

	// Redirect URI du web player
	tidalWebRedirectURI = "https://listen.tidal.com/login/auth"

	tidalClientIDTTL    = 24 * time.Hour
	tidalBundleMaxBytes = 20 * 1024 * 1024 // 20 MB
)

// TidalOAuthCredentials regroupe le client_id et la redirect_uri qui doivent toujours
// être utilisés ensemble (le token exchange doit utiliser la même paire que l'auth URL).
type TidalOAuthCredentials struct {
	ClientID    string
	RedirectURI string
}

var (
	cachedClientID    string
	cachedRedirectURI string
	cachedClientIDAt  time.Time
	clientIDMu        sync.Mutex
)

// GetTidalCredentials retourne la paire (client_id, redirect_uri) à utiliser pour le flow PKCE.
// Priorité : override admin > cache > scraping web player > fallback Android.
func GetTidalCredentials() TidalOAuthCredentials {
	// Override admin (sans verrou — son propre mutex dans proxy_config.go)
	if override := GetTidalClientIDOverride(); override != "" {
		return TidalOAuthCredentials{ClientID: override, RedirectURI: tidalWebRedirectURI}
	}

	clientIDMu.Lock()
	defer clientIDMu.Unlock()

	if cachedClientID != "" && time.Since(cachedClientIDAt) < tidalClientIDTTL {
		return TidalOAuthCredentials{ClientID: cachedClientID, RedirectURI: cachedRedirectURI}
	}

	id, err := scrapeTidalClientID()
	if err != nil || id == "" {
		fmt.Printf("[Tidal] client_id scraping failed (%v) — using Android fallback %s\n", err, tidalAndroidClientID)
		return TidalOAuthCredentials{ClientID: tidalAndroidClientID, RedirectURI: tidalAndroidRedirectURI}
	}

	fmt.Printf("[Tidal] Scraped client_id: %s\n", id)
	cachedClientID = id
	cachedRedirectURI = tidalWebRedirectURI
	cachedClientIDAt = time.Now()
	return TidalOAuthCredentials{ClientID: id, RedirectURI: tidalWebRedirectURI}
}

// GetTidalClientID est un helper pour les cas où seul le client_id est nécessaire.
func GetTidalClientID() string {
	return GetTidalCredentials().ClientID
}

// InvalidateTidalClientIDCache force le re-scraping au prochain appel.
func InvalidateTidalClientIDCache() {
	clientIDMu.Lock()
	cachedClientID = ""
	cachedRedirectURI = ""
	clientIDMu.Unlock()
}

// ─── Scraping ──────────────────────────────────────────────────────────────

func scrapeTidalClientID() (string, error) {
	httpClient := &http.Client{Timeout: 12 * time.Second}

	// ── Étape 1 : récupérer le HTML de listen.tidal.com ─────────────────────
	req, _ := http.NewRequest("GET", "https://listen.tidal.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch listen.tidal.com: %w", err)
	}
	defer resp.Body.Close()
	html, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("read HTML: %w", err)
	}

	// ── Étape 2 : collecter tous les bundles JS ──────────────────────────────
	// Vite génère : <script type="module" crossorigin src="/assets/foo-HASH.js">
	bundleRe := regexp.MustCompile(`src="(/assets/[^"]+\.js)"`)
	bundleMatches := bundleRe.FindAllSubmatch(html, -1)
	if len(bundleMatches) == 0 {
		// Essayer sans leading slash
		bundleRe = regexp.MustCompile(`src="(assets/[^"]+\.js)"`)
		bundleMatches = bundleRe.FindAllSubmatch(html, -1)
	}
	if len(bundleMatches) == 0 {
		return "", fmt.Errorf("no JS bundles found in listen.tidal.com HTML")
	}

	var bundleURLs []string
	for _, m := range bundleMatches {
		path := string(m[1])
		if len(path) > 0 && path[0] == '/' {
			bundleURLs = append(bundleURLs, "https://listen.tidal.com"+path)
		} else {
			bundleURLs = append(bundleURLs, "https://listen.tidal.com/"+path)
		}
	}

	// ── Étape 3 : chercher dans chaque bundle jusqu'à trouver le client_id ──
	var lastErr error
	for _, bundleURL := range bundleURLs {
		id, err := searchBundleForClientID(httpClient, bundleURL, req.Header.Get("User-Agent"))
		if err != nil {
			lastErr = err
			continue
		}
		if id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("clientId not found in %d bundle(s); last error: %v", len(bundleURLs), lastErr)
}

func searchBundleForClientID(httpClient *http.Client, bundleURL, userAgent string) (string, error) {
	req, _ := http.NewRequest("GET", bundleURL, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", bundleURL, err)
	}
	defer resp.Body.Close()
	bundle, err := io.ReadAll(io.LimitReader(resp.Body, tidalBundleMaxBytes))
	if err != nil {
		return "", fmt.Errorf("read bundle: %w", err)
	}

	// ── Stratégie 1 : proximité avec la redirect_uri ─────────────────────────
	// La redirect_uri est toujours un string literal, jamais renommée.
	// Le client_id se trouve dans le même objet de config, juste avant elle.
	// En JS minifié webpack/vite, les slashes sont souvent échappés en \/
	redirectMarkers := [][]byte{
		[]byte(`listen.tidal.com\/login\/auth`), // variante échappée (la plus courante en JS minifié)
		[]byte(`listen.tidal.com/login/auth`),   // variante non-échappée
	}
	for _, redirectMarker := range redirectMarkers {
		idx := bytes.Index(bundle, redirectMarker)
		if idx <= 200 {
			continue
		}
		start := idx - 600
		if start < 0 {
			start = 0
		}
		window := bundle[start:idx]

		// Chercher des string literals alphanumériques de 14–20 chars
		// dans la fenêtre précédant la redirect_uri.
		// On prend le DERNIER (le plus proche de la redirect_uri).
		litRe := regexp.MustCompile(`"([A-Za-z0-9]{14,20})"`)
		matches := litRe.FindAllSubmatch(window, -1)
		for i := len(matches) - 1; i >= 0; i-- {
			candidate := string(matches[i][1])
			// Exclure les faux positifs évidents (hashes hex, mots courants)
			if !isProbablyNotClientID(candidate) {
				return candidate, nil
			}
		}
		// Le marker a été trouvé — inutile de chercher l'autre variante
		// (elles pointent au même endroit dans le bundle).
		break
	}

	// ── Stratégie 2 : patterns explicites (si le bundler préserve le nom) ────
	explicitPatterns := []*regexp.Regexp{
		regexp.MustCompile(`clientId\s*:\s*"([A-Za-z0-9]{14,20})"`),
		regexp.MustCompile(`"clientId"\s*:\s*"([A-Za-z0-9]{14,20})"`),
		regexp.MustCompile(`client_id\s*:\s*"([A-Za-z0-9]{14,20})"`),
	}
	for _, re := range explicitPatterns {
		if m := re.FindSubmatch(bundle); m != nil {
			return string(m[1]), nil
		}
	}

	return "", nil // bundle analysé, rien trouvé — essayer le suivant
}

// isProbablyNotClientID écarte les candidats clairement pas des client_ids.
func isProbablyNotClientID(s string) bool {
	// Hashes git/build hex purs (tout en minuscules/chiffres hex seulement)
	isHex := true
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			isHex = false
			break
		}
	}
	return isHex
}
