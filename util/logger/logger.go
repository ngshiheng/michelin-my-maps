package logger

import (
	"log"
	"time"
)

// A utility function used to time function calls
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("running %s took %s", name, elapsed)
}
