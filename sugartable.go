package tiny

import (
  "sync"
  "time"
  "log"
)

// SugarTable is a table within the cache
type SugarTable struct {
  sync.RWMutex

  // The table's name,
  name string
  // All cached items
  items map[interface{}]*SugarItem

  // Timer responsible for triggering cleanup
  cleanupTimer *time.Timer
  // Current time duration
  cleanupInterval time.Duration

  // The logger used for this table
  logger *log.Logger

  // Callback method triggered when adding a new item to the cache
  addedItem func(item *SugarItem)
}

func (table *SugarTable) addInternal(item *SugarItem) {
  // Careful: do not run this method unless the table-mutex is locked!
  // It will unlock it for the celler before running the ckallbacks and checks
  table.log("Adding item with key", item.key, "and lifeSpan of", item.lifeSpan, "to table", table.name)
  table.items[item.key] = item

  // Cache values so we donot keep blocking the mutex
  expDur := table.cleanupInterval
  addedItem := table.addedItem
  table.Unlock()

  // Trigger callback after adding an item to cache
  if addedItem != nil {
    addedItem(item)
  }

  if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
    // table.expirationCheck()
  }
}

// Value returns an item from the cache and marks it to be kept alive. You can
// pass additional arguments to your DataLoader callback function.
func (table *SugarTable) Value(key interface{}, args ...interface{}) (*SugarItem, error) {
  table.RLock()
  r, ok := table.items[key]
  // loadData := table.loadData
  table.RUnlock()

  if ok {
    // Update access counter and timestamp
    r.KeepAlive()
    return r, nil
  }

  return nil, ErrKeyNotFound
}

// Add adds a key/value pair to the cache
// Parameter key is the item's cache-key
// Parameter lifeSpan determines after which time period without an access the time
// will get removed from the cache.
// Parameter data is the item's value
func (table *SugarTable) Add(key interface{}, lifeSpan time.Duration, data interface{}) *SugarItem {
  item := NewSugarItem(key, lifeSpan, data)

  // Add item to cache
  table.Lock()
  table.addInternal(item)

  return item
}

// Internal logging method for convenience
func (table *SugarTable) log(v ...interface{}) {
  if table.logger == nil {
    return
  }

  table.logger.Println(v...)
}
