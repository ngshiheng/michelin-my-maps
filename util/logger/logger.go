package logger

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// TimeTrack is used to time function calls
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{"name": name}).Infof("running %s took %s", name, elapsed)
}
