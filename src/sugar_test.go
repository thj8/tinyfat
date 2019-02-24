package tiny

import (
  "testing"
  "time"
  "sync"
  "sync/atomic"
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

func TestExist(t *testing.T) {
  // add an expiring item
  table := Sugar("testExist")
  table.Add(k, 0, v)
  // check if it exist
  if !table.Exists(k) {
    t.Error("Error veifying existing data in cache")
  }
}

func TestNotFoundAdd(t *testing.T) {
  table := Sugar("testNotFoundAdd")

  if !table.NotFoundAdd(k, 0, v) {
    t.Error("Error verifying NotFoundAdd, data not in cache")
  }

  if table.NotFoundAdd(k, 0, v) {
    t.Error("Error verifying NotFoundAdd data in cache")
  }
}

func TestNotFoundAddConcurreny(t *testing.T) {
  table := Sugar("TestNotFoundAddConcurreny")

  var finish sync.WaitGroup
  var added int32
  var idle int32

  fn := func(id int) {
    for i := 0; i < 100; i++ {
      if table.NotFoundAdd(i, 0, i+id) {
        atomic.AddInt32(&added, 1)
      } else {
        atomic.AddInt32(&idle, 1)
      }
      time.Sleep(0)
    }
    finish.Done()
  }

  finish.Add(10)
  go fn(0x0000)
  go fn(0x1100)
  go fn(0x2200)
  go fn(0x3300)
  go fn(0x4400)
  go fn(0x5500)
  go fn(0x6600)
  go fn(0x7700)
  go fn(0x8800)
  go fn(0x9900)
  finish.Wait()

  t.Log(added, idle)

  table.Foreach(func(key interface{}, item *SugarItem) {
    v, _ := item.Data().(int)
    k, _ := key.(int)
    t.Logf("%02x   %04x\n", k, v)
  })
}

func TestSugarKeepAlive(t *testing.T) {
  // add an expiring item
  table := Sugar("TestSugarKeepAlive")
  p := table.Add(k, 100 * time.Millisecond, v)

  // keep it alive before it expires
  time.Sleep(50 * time.Millisecond)
  p.KeepAlive()

  // check it still cache after it was initially supposed to expire
  time.Sleep(75 * time.Millisecond)
  if !table.Exists(k) {
    t.Error("Error keeping item alive")
  }

  // check it expires eventally
  time.Sleep(75 * time.Millisecond)
  if table.Exists(k) {
    t.Error("Error expiring item after keeping it alive")
  }
}

func TestDelete(t *testing.T) {
  table := Sugar("TestDelete")
  table.Add(k, 0, v)

  // check it really cached
  p, err := table.Value(k)
  if err != nil || p == nil || p.Data().(string) != v {
    t.Error("Error retrieving data from cache", err)
  }

  // try to delete it
  table.Delete(k)
  // check it really deleted
  p, err = table.Value(k)
  if err == nil || p != nil {
    t.Error("Error deleting item")
  }

  // test err handling
  _, err = table.Delete(k)
  if err == nil {
    t.Error("Excepted error deleting item")
  }
}

func TestFlush(t *testing.T) {
  table := Sugar("TestFlush")
  table.Add(k, 10 * time.Second, v)

  // flush entire table
  table.Flush()

  // try to retrieve the item
  p, err := table.Value(k)
  if p != nil || err == nil {
    t.Error("Error flushing table")
  }

  // make sure there is really noting else left in the cache
  if table.Count() != 0 {
    t.Error("Error verifying count of flushed table")
  }

}
