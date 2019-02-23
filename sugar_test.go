package tiny

import (
	"testing"
	"time"
)

var (
	k = "testkey"
	v = "testvalue"
)

func TestSugar(t *testing.T) {
	// add an expiring item after a non-expiring one to
	// trigger expirationCheck iterating over non-expiring items
	table := Sugar("testCache")
	table.Add(k+"_1", 0*time.Second, v)
	table.Add(k+"_2", 1*time.Second, v)

	// check if both items are still there
	p, err := table.Value(k + "_1")
	if err != nil || p == nil || p.Data().(string) != v {
		t.Error("Error retrieving non expiring data from cache", err)
	}
  p, err = table.Value(k + "_2")
  if err != nil || p == nil || p.Data().(string) != v {
    t.Error("Errir retrieving data from cache", err)
  }

  // sanity check
  if p.AccessCount() != 1 {
    t.Error("Error getting correct access count")
  }
  if p.LifeSpan() != 1 * time.Second {
    t.Error("Error getting correct life-span")
  }
  if p.AccessedOn().Unix() == 0 {
    t.Error("Error getting eccess time")
  }
  if p.CreatedOn().Unix() == 0 {
    t.Error("Error getting creation time")
  }
}
