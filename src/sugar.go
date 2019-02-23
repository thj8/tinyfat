package tiny

import (
  "sync"
)

var (
  cache = make(map[string]*SugarTable)
  mutex sync.RWMutex
)

// Sugar returns the existing cache table with given name or create a new one
// if the table does not exist yet
func Sugar(table string) *SugarTable {
  mutex.RLock()
  t, ok := cache[table]
  mutex.RUnlock()

  if !ok {
    mutex.Lock()
    t, ok = cache[table]
    if !ok {
      t = &SugarTable{
        name: table,
        items: make(map[interface{}]*SugarItem),
      }
      cache[table] = t
    }
    mutex.Unlock()
  }

  return t
}
