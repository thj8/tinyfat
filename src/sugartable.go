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

  // Callback method triggered when trying to load a non-existing key.
  loadData func(key interface{}, args ...interface{}) *SugarItem
  // Callback method triggered when adding a new item to the cache
  addedItem func(item *SugarItem)
  // Callback method triggered before deleting an item from the cache.
  aboutToDeleteItem func(item *SugarItem)
}

func (table *SugarTable) expirationCheck() {
  table.Lock()
  if table.cleanupTimer != nil {
    table.cleanupTimer.Stop()
  }

  if table.cleanupInterval > 0 {
    table.log("Expiration check triggered agter", table.cleanupInterval, "for table", table.name)
  } else {
    table.log("Expiration check installed for table", table.name)
  }

  // To be more accurate with timers, we would need to update now on every
  // loop iteration. Not sure it's really efficient though
  now := time.Now()
  smallestDuration := 0 * time.Second
  for key, item := range table.items {
    item.RLock()
    lifeSpan := item.lifeSpan
    accessedOn := item.accessedOn
    item.RUnlock()

    if lifeSpan == 0 {
      continue
    }

    if now.Sub(accessedOn) >= lifeSpan {
      // Item has excessed its lifeSpan.
      table.deleteInternal(key)
    } else {
      // Find the item chronologically closest to its end-of-lifespan.
      if smallestDuration == 0 || lifeSpan - now.Sub(accessedOn) < smallestDuration {
        smallestDuration = lifeSpan - now.Sub(accessedOn)
      }
    }
  }

  // Setup the interval for the next cleanup run.
  table.cleanupInterval = smallestDuration
  if smallestDuration > 0 {
    table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
      go table.expirationCheck()
    })
  }

  table.Unlock()
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

  // If we haven't set up any expiration check timer or found a more imminent item.
  if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
     table.expirationCheck()
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

func (table *SugarTable) deleteInternal(key interface{}) (*SugarItem, error) {
  r, ok := table.items[key]
  if !ok {
    return nil, ErrKeyNotFound
  }

  // Cache value so we don't keep blocking the mutex.
  aboutToDeleteItem := table.aboutToDeleteItem
  table.Unlock()

  // Trigger callbacks before deleting an item from cache.
  if aboutToDeleteItem != nil {
    aboutToDeleteItem(r)
  }

  r.RLock()
  defer r.RUnlock()
  if r.aboutToExpire != nil {
    r.aboutToExpire(key)
  }

  table.Lock()
  table.log("Deleting iteam with key", key, "created on", r.createdOn,
    "and hit", r.accessCount, "times from table", table.name)
  delete(table.items, key)
  return r, nil
}

// Delete an item from the cache.
func (table *SugarTable) Delete(key interface{}) (*SugarItem, error) {
  table.Lock()
  defer table.Unlock()

  return table.deleteInternal(key)
}

// Internal logging method for convenience.
func (table *SugarTable) log(v ...interface{}) {
  if table.logger == nil {
    return
  }

  table.logger.Println(v...)
}