package macaco

import (
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/rainycape/otto"
)

var (
	maxAgeRe = regexp.MustCompile("max-age=(\\d+)")
)

type diskEntry struct {
	URL        string
	Data       []byte
	StatusCode int
	Header     http.Header
	Expires    time.Time
}

type scriptEntry struct {
	script  *otto.Script
	expires time.Time
}

type cache struct {
	sync.RWMutex
	scripts map[string]*scriptEntry
}

func newCache() *cache {
	c := new(cache)
	c.scripts = make(map[string]*scriptEntry)
	return c
}

func (c *cache) root() (string, error) {
	dir, err := macacoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cache"), nil
}

func (c *cache) cachePath(url string) (string, error) {
	r, err := c.root()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	h.Write([]byte(url))
	base := hex.EncodeToString(h.Sum(nil))
	return filepath.Join(r, "http", base), nil
}

func (c *cache) parseExpiration(headers http.Header) time.Time {
	var expires time.Time
	cacheControl := headers.Get("Cache-Control")
	// TODO: must-revalidate
	if cacheControl != "" {
		m := maxAgeRe.FindStringSubmatch(cacheControl)
		if len(m) > 1 {
			duration, _ := strconv.Atoi(m[1])
			if duration > 0 {
				expires = time.Now().Add(time.Duration(duration) * time.Second)
			}
		}
	}
	if expires.IsZero() {
		// Check Expires header
		expires, _ = time.Parse(time.RFC1123, headers.Get("Expires"))
	}
	return expires
}

func (c *cache) cacheData(url string, body []byte, resp *http.Response) error {
	expires := c.parseExpiration(resp.Header)
	if expires.IsZero() {
		return nil
	}
	p, err := c.cachePath(url)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	entry := &diskEntry{
		URL:        resp.Request.URL.String(),
		Data:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Expires:    expires,
	}
	if err := gob.NewEncoder(f).Encode(entry); err != nil {
		return err
	}
	return f.Close()
}

func (c *cache) cachedEntry(url string) (*diskEntry, error) {
	p, err := c.cachePath(url)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var de *diskEntry
	if err := gob.NewDecoder(f).Decode(&de); err != nil {
		return nil, err
	}
	return de, nil
}

func (c *cache) cacheScript(url string, data []byte, resp *http.Response, script *otto.Script, entry *diskEntry) error {
	var expires time.Time
	if entry != nil {
		expires = entry.Expires
	} else if resp != nil {
		expires = c.parseExpiration(resp.Header)
	}
	if expires.IsZero() {
		// Not cacheable
		return nil
	}
	c.Lock()
	c.scripts[url] = &scriptEntry{
		script:  script,
		expires: expires,
	}
	c.Unlock()
	if entry == nil {
		return c.cacheData(url, data, resp)
	}
	return nil
}

func (c *cache) getCachedScript(url string) (*diskEntry, *otto.Script) {
	c.RLock()
	entry := c.scripts[url]
	c.RUnlock()
	if entry != nil {
		if time.Since(entry.expires) < 0 {
			return nil, entry.script
		}
		// In-Memory entry is expired, so we can assume the
		// disk entry is also expired.
		c.Lock()
		delete(c.scripts, url)
		c.Unlock()
		return nil, nil
	}
	de, _ := c.cachedEntry(url)
	return de, nil
}

func init() {
	gob.Register(diskEntry{})
}
