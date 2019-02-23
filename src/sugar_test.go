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

func TestSugarExpire(t *testing.T) {
  table := Sugar("testCache")
  table.Add(k+"_1", 100*time.Millisecond, v+"_1")
  table.Add(k+"_2", 125*time.Millisecond, v+"_2")

  time.Sleep(75 * time.Millisecond)

  // check key 1 is still alive
  _, err := table.Value(k + "_1")
  if err != nil {
    t.Error("Error retrieving value from cache:", err)
  }

  time.Sleep(75 * time.Millisecond)

  // check key 1 agin, it should still be alive since we just accessed it
  _, err = table.Value(k + "_1")
  if err != nil {
    t.Error("Error retrieving value from cache:", err)
  }

  _, err = table.Value(k + "_2")
  if err == nil {
    t.Error("Found key which should have been expired by now")
  }

}
