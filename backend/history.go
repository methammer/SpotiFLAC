package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

type HistoryItem struct {
	ID          string `json:"id"`
	SpotifyID   string `json:"spotify_id"`
	Title       string `json:"title"`
	Artists     string `json:"artists"`
	Album       string `json:"album"`
	DurationStr string `json:"duration_str"`
	CoverURL    string `json:"cover_url"`
	Quality     string `json:"quality"`
	Format      string `json:"format"`
	Path        string `json:"path"`
	Timestamp   int64  `json:"timestamp"`
}

type FetchHistoryItem struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Info      string `json:"info"`
	Image     string `json:"image"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// ─────────────────────────────────────────────────────────────────────────────
// State
// ─────────────────────────────────────────────────────────────────────────────

var (
	historyDB        *bolt.DB
	historyDisabled  bool          // true si toutes les tentatives ont échoué
	historyConfigDir string        // chemin défini par InitHistoryDBAt
	historyMu        sync.Mutex    // protège l'init lazy
)

const (
	historyBucket      = "DownloadHistory"
	fetchHistoryBucket = "FetchHistory"
	maxHistory         = 10000
)

// ─────────────────────────────────────────────────────────────────────────────
// Init
// ─────────────────────────────────────────────────────────────────────────────

// InitHistoryDBAt ouvre (ou crée) history.db dans configDir.
// Réessaie jusqu'à 3 fois avec timeout croissant pour survivre aux
// redémarrages rapides Docker qui laissent un flock BoltDB en suspens.
func InitHistoryDBAt(configDir string) error {
	historyMu.Lock()
	defer historyMu.Unlock()

	historyConfigDir = configDir

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	dbPath := filepath.Join(configDir, "history.db")

	// Tentatives avec timeouts croissants : 3s → 5s → 8s
	timeouts := []time.Duration{3 * time.Second, 5 * time.Second, 8 * time.Second}
	var lastErr error

	for attempt, timeout := range timeouts {
		if attempt > 0 {
			fmt.Printf("[History] Retry %d/%d opening history.db (timeout: %v)...\n",
				attempt, len(timeouts)-1, timeout)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: timeout})
		if err != nil {
			lastErr = err
			fmt.Printf("[History] Attempt %d failed: %v\n", attempt+1, err)
			continue
		}

		// Créer les buckets si nécessaire
		err = db.Update(func(tx *bolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists([]byte(historyBucket)); err != nil {
				return err
			}
			if _, err := tx.CreateBucketIfNotExists([]byte(fetchHistoryBucket)); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			db.Close()
			lastErr = err
			continue
		}

		historyDB = db
		historyDisabled = false
		fmt.Printf("[History] history.db opened: %s\n", dbPath)
		return nil
	}

	// Toutes les tentatives ont échoué → mode dégradé
	historyDisabled = true
	fmt.Printf("[History] WARNING: history DB unavailable after %d attempts: %v\n",
		len(timeouts), lastErr)
	return fmt.Errorf("history DB unavailable: %w", lastErr)
}

// InitHistoryDB est l'ancienne API (compatibilité Wails desktop).
// En mode web, préférer InitHistoryDBAt.
func InitHistoryDB(appName string) error {
	appDir, err := GetFFmpegDir()
	if err != nil {
		return err
	}
	return InitHistoryDBAt(appDir)
}

func CloseHistoryDB() {
	historyMu.Lock()
	defer historyMu.Unlock()
	if historyDB != nil {
		historyDB.Close()
		historyDB = nil
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper interne : obtenir ou ré-initialiser historyDB
// ─────────────────────────────────────────────────────────────────────────────

// getHistoryDB retourne historyDB prêt à l'emploi.
// Si historyDB est nil (init échouée au démarrage), tente une re-init.
// Retourne une erreur si définitivement indisponible.
func getHistoryDB() (*bolt.DB, error) {
	historyMu.Lock()
	defer historyMu.Unlock()

	if historyDisabled {
		return nil, fmt.Errorf("history DB is disabled (failed to open at startup)")
	}
	if historyDB != nil {
		return historyDB, nil
	}

	// Tentative de ré-init lazy avec le bon chemin
	if historyConfigDir == "" {
		return nil, fmt.Errorf("history DB not initialized")
	}

	dbPath := filepath.Join(historyConfigDir, "history.db")
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		historyDisabled = true
		return nil, fmt.Errorf("history DB re-init failed: %w", err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(historyBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(fetchHistoryBucket)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("history DB bucket init failed: %w", err)
	}

	historyDB = db
	fmt.Printf("[History] history.db re-initialized: %s\n", dbPath)
	return historyDB, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Download History
// ─────────────────────────────────────────────────────────────────────────────

func AddHistoryItem(item HistoryItem, appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		fmt.Printf("[History] AddHistoryItem skipped: %v\n", err)
		return nil // non-fatal : l'historique est optionnel
	}
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(historyBucket))
		if err != nil {
			return err
		}
		id, _ := b.NextSequence()
		item.ID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
		item.Timestamp = time.Now().Unix()

		buf, err := json.Marshal(item)
		if err != nil {
			return err
		}

		// Élagage si limite atteinte
		if b.Stats().KeyN >= maxHistory {
			c := b.Cursor()
			toDelete := maxHistory / 20
			if toDelete < 1 {
				toDelete = 1
			}
			count := 0
			for k, _ := c.First(); k != nil && count < toDelete; k, _ = c.Next() {
				b.Delete(k)
				count++
			}
		}

		return b.Put([]byte(item.ID), buf)
	})
}

func GetHistoryItems(appName string) ([]HistoryItem, error) {
	db, err := getHistoryDB()
	if err != nil {
		return []HistoryItem{}, nil // retourner slice vide plutôt qu'erreur
	}
	var items []HistoryItem
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(historyBucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var item HistoryItem
			if err := json.Unmarshal(v, &item); err == nil {
				items = append(items, item)
			}
		}
		return nil
	})
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp > items[j].Timestamp
	})
	return items, err
}

func ClearHistory(appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(historyBucket))
	})
}

func DeleteHistoryItem(id string, appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(historyBucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Fetch History
// ─────────────────────────────────────────────────────────────────────────────

func AddFetchHistoryItem(item FetchHistoryItem, appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		fmt.Printf("[History] AddFetchHistoryItem skipped: %v\n", err)
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(fetchHistoryBucket))
		if err != nil {
			return err
		}

		// Dédupliquer par URL+Type
		if item.URL != "" {
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var existing FetchHistoryItem
				if err := json.Unmarshal(v, &existing); err == nil {
					if existing.URL == item.URL && existing.Type == item.Type {
						b.Delete(k)
					}
				}
			}
		}

		id, _ := b.NextSequence()
		item.ID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
		item.Timestamp = time.Now().Unix()

		buf, err := json.Marshal(item)
		if err != nil {
			return err
		}

		if b.Stats().KeyN >= maxHistory {
			c := b.Cursor()
			toDelete := maxHistory / 20
			if toDelete < 1 {
				toDelete = 1
			}
			count := 0
			for k, _ := c.First(); k != nil && count < toDelete; k, _ = c.Next() {
				b.Delete(k)
				count++
			}
		}

		return b.Put([]byte(item.ID), buf)
	})
}

func GetFetchHistoryItems(appName string) ([]FetchHistoryItem, error) {
	db, err := getHistoryDB()
	if err != nil {
		return []FetchHistoryItem{}, nil
	}
	var items []FetchHistoryItem
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(fetchHistoryBucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var item FetchHistoryItem
			if err := json.Unmarshal(v, &item); err == nil {
				items = append(items, item)
			}
		}
		return nil
	})
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp > items[j].Timestamp
	})
	return items, err
}

func ClearFetchHistory(appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(fetchHistoryBucket))
	})
}

func ClearFetchHistoryByType(itemType string, appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(fetchHistoryBucket))
		if b == nil {
			return nil
		}
		var keysToDelete [][]byte
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var item FetchHistoryItem
			if err := json.Unmarshal(v, &item); err == nil && item.Type == itemType {
				keysToDelete = append(keysToDelete, k)
			}
		}
		for _, k := range keysToDelete {
			b.Delete(k)
		}
		return nil
	})
}

func DeleteFetchHistoryItem(id string, appName string) error {
	db, err := getHistoryDB()
	if err != nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(fetchHistoryBucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	})
}
