package cmd

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/thisisbalu/nemi/internal/build"
)

var serveDrafts bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the site locally with live reload",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolVarP(&serveDrafts, "drafts", "D", false, "include draft pages")
}

func runServe(_ *cobra.Command, _ []string) error {
	if _, err := build.Run(true, serveDrafts); err != nil {
		fmt.Printf("build error: %v\n", err)
	}

	var (
		clients = make(map[chan string]struct{})
		mu      sync.Mutex
	)

	addClient := func(ch chan string) {
		mu.Lock()
		clients[ch] = struct{}{}
		mu.Unlock()
	}
	removeClient := func(ch chan string) {
		mu.Lock()
		delete(clients, ch)
		mu.Unlock()
	}
	broadcast := func(msg string) {
		mu.Lock()
		for ch := range clients {
			select {
			case ch <- msg:
			default:
			}
		}
		mu.Unlock()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, dir := range []string{"content", "layouts", "static", "data"} {
		_ = watchDirRecursive(watcher, dir)
	}

	go func() {
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
					if debounce != nil {
						debounce.Stop()
					}
					debounce = time.AfterFunc(150*time.Millisecond, func() {
						start := time.Now()
						stats, err := build.Run(true, serveDrafts)
						if err != nil {
							fmt.Printf("→ build error: %v\n", err)
							broadcast("error:" + strings.ReplaceAll(err.Error(), "\n", " "))
							return
						}
						fmt.Printf("→ rebuilt %d pages in %s\n", stats.Pages, time.Since(start).Round(time.Millisecond))
						broadcast("reload")
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("watcher: %v\n", err)
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/__nemi_reload", func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")

		ch := make(chan string, 1)
		addClient(ch)
		defer removeClient(ch)

		flusher, canFlush := w.(http.Flusher)
		for {
			select {
			case msg := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				if canFlush {
					flusher.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.Handle("/", http.FileServer(http.Dir("public")))

	fmt.Println("Nemi dev server → http://localhost:3000")
	fmt.Println("Watching for changes... (Ctrl+C to stop)")
	return http.ListenAndServe(":3000", mux)
}

func watchDirRecursive(w *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		return w.Add(path)
	})
}
