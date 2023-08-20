package logger

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// TimeTrack tracks the time elapsed for a function call and logs the duration.
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{
		"name":    name,
		"elapsed": elapsed,
	}).Infof("function %s took %s", name, elapsed)
}
