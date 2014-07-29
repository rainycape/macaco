package macaco

import (
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"net/http"
	"os"
	"os/user"
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
	Data    []byte
	Headers http.Header
	Expires time.Time
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
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".macaco", "cache"), nil
}

func (c *cache) scriptPath(url string) (string, error) {
	r, err := c.root()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	h.Write([]byte(url))
	base := hex.EncodeToString(h.Sum(nil))
	return filepath.Join(r, "scripts", base), nil
}

func (c *cache) cacheScript(url string, data []byte, headers http.Header, script *otto.Script, expires time.Time) error {
	onDisk := expires.IsZero()
	if onDisk {
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
	if onDisk {
		p, err := c.scriptPath(url)
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
		if err := gob.NewEncoder(f).Encode(&diskEntry{Data: data, Headers: headers, Expires: expires}); err != nil {
			return err
		}
		return f.Close()
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
	if p, _ := c.scriptPath(url); p != "" {
		if f, _ := os.Open(p); f != nil {
			defer f.Close()
			var de *diskEntry
			if gob.NewDecoder(f).Decode(&de) == nil {
				return de, nil
			}
		}
	}
	return nil, nil
}

func init() {
	gob.Register(diskEntry{})
}
