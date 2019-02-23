package tiny

import (
  "sync"
  "time"
)

// SugarItem is an individual cache item
// Parameter data contains the user-set value in the cache
type SugarItem struct {
  sync.RWMutex

  // The item's key
  key interface{}
  // The item's value
  data interface{}
  // How long will the item live in the cache when not being accessed/kept alive.
  lifeSpan time.Duration

  // Creation timestamp
  createdOn time.Time
  // Last access timestamp
  accessedOn time.Time
  // How often the item was accessed
  accessCount int64

  // Callback method triggered right before removing the item from the cache
  aboutToExpire func(key interface{})
}

// NewSugarItem returns a newly created SugarItem.
// Parameter key is the item's cache-key
// Parameter lifeSpan determines after which time period without an access the item
// will get removed from the cached.
// Paramter data is the item's value
func NewSugarItem(key interface{}, lifeSpan time.Duration, data interface{}) * SugarItem {
  t := time.Now()
  return &SugarItem{
    key:        key,
    lifeSpan:   lifeSpan,
    createdOn:  t,
    accessedOn: t,
    accessCount: 0,
    aboutToExpire: nil,
    data:       data,
  }
}

// KeepAlive marks an item to be kept for another expireDuration peroid
func (item *SugarItem) KeepAlive() {
  item.Lock()
  defer item.Unlock()
  item.accessedOn = time.Now()
  item.accessCount++
}

// Key returns the key of this cached item.
func (item *SugarItem) Key() interface{} {
  return item.key
}

// Data returns the data of this cached item.
func (item *SugarItem) Data() interface{} {
  return item.data
}

// LifeSpan returns this item's expiration duration.
func (item *SugarItem) LifeSpan() time.Duration {
  // immutable
  return item.lifeSpan
}

// AccessedOn returns when this item was last accessed.
func (item *SugarItem) AccessedOn() time.Time {
  item.RLock()
  defer item.RUnlock()
  return item.accessedOn
}

// CreatedOn returns when this item was added to the cached
func (item *SugarItem) CreatedOn() time.Time {
  // immutable
  return item.createdOn
}

// AccessCount returns how often this item has been accessed.
func (item *SugarItem) AccessCount() int64 {
  item.RLock()
  defer item.RUnlock()
  return item.accessCount
}

// SetAboutToExpireCallback configures a callback, which will be called right
// bdfore the item is about to be removed from the cache.
func (item *SugarItem) SetAboutToExpireCallback(f func(interface {})) {
  item.Lock()
  defer item.Unlock()
  item.aboutToExpire = f
}
