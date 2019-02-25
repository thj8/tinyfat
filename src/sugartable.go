package tiny

import (
  "sync"
  "time"
  "log"
  "sort"
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

// Foreach all items
func (table *SugarTable) Foreach(trans func(key interface{}, item *SugarItem)) {
  table.RLock()
  defer table.RUnlock()

  for k, v := range table.items {
    trans(k,v)
  }
}

// SetDataLoader configures a data-loader callback, which will be called when
// trying to access a non-existing key.The key and 0...n additional arguments
// are passed to the callback function.
func (table *SugarTable) SetDataLoader(f func(interface{}, ...interface{}) *SugarItem) {
  table.Lock()
  defer table.Unlock()

  table.loadData = f
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
  loadData := table.loadData
  table.RUnlock()

  if ok {
    // Update access counter and timestamp
    r.KeepAlive()
    return r, nil
  }

  // Item doesn't exist in cache. Try and fetch it with a data-loader.
  if loadData != nil {
    item := loadData(key, args...)
    if item != nil {
      table.Add(key, item.lifeSpan, item.data)
      return item, nil
    }

    return nil, ErrKeyNotFound
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

// Exists returns whether an item exists in the cache.Unlike the Value method
// Exists neither tries to fetch data via lodaData callback nor does it 
// keep the item alive in the cache.
func (table *SugarTable) Exists(key interface{}) bool {
  table.RLock()
  defer table.RUnlock()

  _, ok := table.items[key]
  return ok
}

// NotFoundAdd tests whether an item not found in the cache.Unlike the Exists
// method this also adds data if the key cloud not be found.
func (table *SugarTable) NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool {
  table.Lock()

  if _, ok := table.items[key]; ok {
    table.Unlock()
    return false
  }

  item := NewSugarItem(key, lifeSpan, data)
  table.addInternal(item)

  return true
}

// Flush deletes all item from the cache.
func (table *SugarTable) Flush() {
  table.Lock()
  defer table.Unlock()

  table.log("Flushing the tale", table.name)

  table.items = make(map[interface{}]*SugarItem)
  table.cleanupInterval = 0
  if table.cleanupTimer != nil {
    table.cleanupTimer.Stop()
  }
}

// Count returns how many items are currently stored in the cache.
func (table *SugarTable) Count() int {
  table.RLock()
  defer table.RUnlock()

  return len(table.items)
}

// SugarItemPair maps key to access counter
type SugarItemPair struct {
  Key     interface{}
  AccessCount   int64
}

// SugarItemPairList is a slice of SugarItemPairs that implements sort.
// Interface to sort by AccessCount
type SugarItemPairList []SugarItemPair

func (p SugarItemPairList) Swap(i, j int)   { p[i], p[j] = p[j], p[i] }
func (p SugarItemPairList) Len() int        { return len(p)}
func (p SugarItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }

// MostAccessed returs the most accessed items in this cache table
func (table *SugarTable) MostAccessed(count int64) []*SugarItem {
  table.RLock()
  defer table.RUnlock()

  p := make(SugarItemPairList, len(table.items))
  i := 0
  for k, v := range table.items {
    p[i] = SugarItemPair{k, v.accessCount}
    i++
  }
  sort.Sort(p)

  var r []*SugarItem
  c := int64(0)
  for _, v := range p {
    if c >= count {
      break
    }

    item, ok := table.items[v.Key]
    if ok {
      r = append(r, item)
    }
    c++
  }

  return r
}


// Internal logging method for convenience.
func (table *SugarTable) log(v ...interface{}) {
  if table.logger == nil {
    return
  }

  table.logger.Println(v...)
}

