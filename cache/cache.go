package cache

import (
	"errors"
	"os"
	"path"
	"time"
)

const prefix = "via-cache-"

var ttls = map[string]time.Time{}

func Store(name string, data []byte, ttl time.Duration) error {
	p := path.Join(os.TempDir(), name)

	err := os.WriteFile(p, data, 0666)
	if err != nil {
		return err
	}

	ttls[name] = time.Now().Add(ttl)
	return nil
}

func Load(name string) ([]byte, error) {
	ttl, ok := ttls[name]
	if ok && time.Now().After(ttl) {
		return nil, errors.New("time exceeded")
	}

	p := path.Join(os.TempDir(), name)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return data, nil
}
