package channel

import (
	"strconv"
	"sync/atomic"
)

var globalNumber int64 = 0

// generateID gerenates id that sufficient to represent unique
// identifier to be used on in memory channel
func generateID() string {
	i := atomic.AddInt64(&globalNumber, 1)
	return strconv.FormatInt(i, 10)
}
