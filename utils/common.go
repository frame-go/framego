package utils

import (
	"math/rand"
	"os"
	"time"
)

// InitRand initializes random seed by PID and current nano timestamp.
// The random seed should be initialized before any call to rand API.
func InitRand() {
	pid := int64(os.Getpid())
	now := time.Now().UnixNano()
	seed := (pid << 56) ^ now
	rand.Seed(seed)
}
