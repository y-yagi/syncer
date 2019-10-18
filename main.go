package main

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"path/filepath"

	"github.com/y-yagi/goext/osext"
	"gopkg.in/fsnotify.v1"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	logger, err := syslog.New(syslog.LOG_WARNING|syslog.LOG_DAEMON, "syncer")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	targets := map[string]string{
		"/home/yaginuma/.histfile": "/home/yaginuma/Dropbox/backup/.histfile",
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				dest := targets[event.Name]
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := copyFile(event.Name, dest)
					if err != nil {
						logger.Err(fmt.Sprintf("%v", err))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Err(fmt.Sprintf("%v", err))
			}
		}
	}()

	for s, _ := range targets {
		err = watcher.Add(s)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}

func copyFile(source string, dest string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	if osext.IsExist(dest) {
		err = os.Remove(dest)
		if err != nil {
			return err
		}
	} else {
		err = os.MkdirAll(filepath.Dir(dest), 0666)
		if err != nil {
			return err
		}
	}

	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}
