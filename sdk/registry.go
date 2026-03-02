package sdk

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	mu      sync.RWMutex
	entries []entry
)

type entry struct {
	name string
	mod  Exploit
}

// Register adds an exploit to the global registry.
// The name is derived from the caller's file path relative to modules/.
// callerSkip controls stack depth: use 0 when calling from init() directly,
// use 1 when called through an intermediary (like sdk.Register wrapper).
func Register(mod Exploit) {
	register(mod, 2)
}

func register(mod Exploit, skip int) {
	name := callerModuleName(skip)
	mu.Lock()
	defer mu.Unlock()
	for _, e := range entries {
		if e.name == name {
			panic(fmt.Sprintf("exploit %q already registered", name))
		}
	}
	entries = append(entries, entry{name, mod})
}

// Get returns an exploit by full path or short name.
func Get(name string) Exploit {
	mu.RLock()
	defer mu.RUnlock()

	lower := strings.ToLower(name)
	for _, e := range entries {
		if e.name == name {
			return e.mod
		}
	}
	for _, e := range entries {
		if strings.ToLower(filepath.Base(e.name)) == lower {
			return e.mod
		}
	}
	return nil
}

// NameOf returns the registered name of an exploit.
func NameOf(mod Exploit) string {
	mu.RLock()
	defer mu.RUnlock()
	for _, e := range entries {
		if e.mod == mod {
			return e.name
		}
	}
	return "unknown"
}

// List returns all registered exploits in registration order.
func List() []Exploit {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Exploit, len(entries))
	for i, e := range entries {
		result[i] = e.mod
	}
	return result
}

// Names returns all registered exploit names in order.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.name
	}
	return result
}

// Search returns exploits matching query against name, description, or CVEs.
func Search(query string) []Exploit {
	mu.RLock()
	defer mu.RUnlock()
	q := strings.ToLower(query)
	var result []Exploit
	for _, e := range entries {
		info := e.mod.Info()
		if strings.Contains(strings.ToLower(e.name), q) ||
			strings.Contains(strings.ToLower(info.Description), q) ||
			strings.Contains(strings.ToLower(strings.Join(info.CVEs(), " ")), q) {
			result = append(result, e.mod)
		}
	}
	return result
}

// callerModuleName derives the exploit path from the caller's file location.
// skip=2 means: callerModuleName -> Register -> init().
func callerModuleName(skip int) string {
	_, file, _, ok := runtime.Caller(skip + 1)
	if !ok {
		panic("sdk.Register: cannot determine caller")
	}
	const marker = "modules/"
	if idx := strings.LastIndex(file, marker); idx != -1 {
		rel := file[idx+len(marker):]
		return strings.TrimSuffix(rel, filepath.Ext(rel))
	}
	return strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
}
