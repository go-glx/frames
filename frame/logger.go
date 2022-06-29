package frame

import "log"

type fallbackLogger struct {
}

func (f *fallbackLogger) Error(err error) {
	log.Println(err.Error())
}

