package boslib

import "github.com/sirupsen/logrus"

var log *logrus.Logger

func init() {
	log = logrus.New()
}

func SetLevel(l logrus.Level) {
	log.SetLevel(l)
}
