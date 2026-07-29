package inklessmcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// previewEntry holds a dry-run merge result for a later commit call.
type previewEntry struct {
	Expires   time.Time
	SiteID    string
	ArticleID uint
	Body      map[string]any
}

// previewStore is an in-process TTL map (stateless MCP-friendly app handle).
type previewStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	byID map[string]previewEntry
}

func newPreviewStore(ttl time.Duration) *previewStore {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &previewStore{ttl: ttl, byID: map[string]previewEntry{}}
}

func (s *previewStore) Put(siteID string, articleID uint, body map[string]any) (string, error) {
	id, err := randomID("pv_")
	if err != nil {
		return "", err
	}
	// shallow copy body map
	cp := make(map[string]any, len(body))
	for k, v := range body {
		cp[k] = v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.byID[id] = previewEntry{
		Expires:   time.Now().Add(s.ttl),
		SiteID:    siteID,
		ArticleID: articleID,
		Body:      cp,
	}
	return id, nil
}

func (s *previewStore) Take(id, siteID string, articleID uint) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	e, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("preview_handle %q not found or expired", id)
	}
	if e.SiteID != siteID || e.ArticleID != articleID {
		return nil, fmt.Errorf("preview_handle does not match site_id/article id")
	}
	delete(s.byID, id)
	return e.Body, nil
}

func (s *previewStore) gcLocked() {
	now := time.Now()
	for id, e := range s.byID {
		if now.After(e.Expires) {
			delete(s.byID, id)
		}
	}
}

func randomID(prefix string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}
