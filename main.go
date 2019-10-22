package main

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"path/filepath"
	"time"

	"github.com/y-yagi/configure"
	"github.com/y-yagi/goext/osext"
	"gopkg.in/fsnotify.v1"
)

const cmd = "syncer"

type Path struct {
	From string
	To   string
}
type config struct {
	Paths []Path `toml:"path"`
}

var cfg config

func init() {
	err := configure.Load(cmd, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Paths) == 0 {
		fmt.Fprintf(os.Stderr, "Please specify paths.\n")
		os.Exit(1)
	}
}

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

	changed := []string{}
	paths := map[string]string{}
	syncDuration := 10 * time.Minute

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					changed = append(changed, event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Err(fmt.Sprintf("%v", err))
			case <-time.After(syncDuration):
				for _, src := range changed {
					err := copyFile(src, paths[src])
					if err != nil {
						logger.Err(fmt.Sprintf("%v", err))
					}
				}
				changed = []string{}
			}
		}
	}()

	for _, p := range cfg.Paths {
		err = watcher.Add(p.From)
		if err != nil {
			log.Fatal(err)
		}
		paths[p.From] = p.To
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
