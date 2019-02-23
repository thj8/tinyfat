package tiny

import (
  "errors"
)

var  (
  // ErrKeyNotFound gets returned when a specific key couldn't be found
  ErrKeyNotFound = errors.New("Key not found in cache")
)
