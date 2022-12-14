package logger

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// TimeTrack tracks the time elapsed for a function call
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{"name": name}).Infof("running %s took %s", name, elapsed)
}
