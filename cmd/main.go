package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syncai/internal/version"
	"syscall"
	"time"

	"syncai/internal/config"
	"syncai/internal/selfupdate"
	"syncai/internal/syncai"
	"syncai/internal/util"
)

func main() {
	var cfgPath string
	var doSelfUpdate bool
	flag.StringVar(&cfgPath, "config", "syncai.json", "path to configuration file")
	flag.BoolVar(&doSelfUpdate, "self-update", false, "update SyncAI to the latest released version")
	flag.Parse()

	if doSelfUpdate {
		if err := selfupdate.Run(); err != nil {
			log.Fatalf("self-update failed: %v", err)
		}
		fmt.Println("SyncAI updated successfully. Restart if it was running.")
		return
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("SyncAI %s\nGitHub: https://github.com/flowmitry/syncai/\n\n", version.Version())
	sync := syncai.New(cfg)
	initialSync(cfg, sync)

	filesState := buildFilesState(cfg)
	scan := func() {
		newState := make(map[string]time.Time)
		for _, agent := range cfg.Agents {
			for _, path := range agent.Files() {
				fi, err := os.Stat(path)
				if err != nil {
					if os.IsNotExist(err) {
						// File may not exist yet; silently ignore
						continue
					}
					log.Printf("file error %s: %v", path, err)
					continue
				}
				modT := fi.ModTime()
				hash, _ := util.FileHash(path)
				newState[path] = modT
				prev, ok := filesState[path]
				if !ok || hash != prev {
					if ok {
						log.Printf("Detected change in %s (%s), syncing...", path, agent.Name)
					} else {
						log.Printf("Detected new file %s (%s), syncing...", path, agent.Name)
					}
					files, err := sync.Sync(path)
					if err != nil {
						log.Printf("sync error: %v", err)
					}
					filesState[path] = hash
					for _, newPath := range files {
						filesState[newPath], _ = util.FileHash(newPath)
					}
				}
			}
		}
		for path := range filesState {
			if _, ok := newState[path]; !ok {
				if err := sync.Delete(path); err != nil {
					log.Printf("delete error: %v", err)
				} else {
					log.Printf("Deleted missing file across agents: %s", path)
					delete(filesState, path)
				}
			}
		}
	}

	ticker := time.NewTicker(cfg.Interval())
	defer ticker.Stop()

	// Handle OS signals to terminate gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			scan()
		case <-sigCh:
			log.Println("Exiting SyncAI...")
			return
		}
	}
}

// Initial sync: pick the newest version among agents for each logical file (by kind+stem) and propagate it
func initialSync(cfg config.Config, sync *syncai.SyncAI) {
	type newest struct {
		path string
		mod  time.Time
	}
	latest := make(map[string]newest)

	for _, agent := range cfg.Agents {
		for _, path := range agent.Files() {
			fi, err := os.Stat(path)
			if err != nil {
				// File might not exist yet; skip
				continue
			}
			modT := fi.ModTime()

			_, kind, stem := sync.Identify(path)
			if kind == syncai.KindUnknown {
				continue
			}

			key := string(kind) + "|" + stem
			if cur, ok := latest[key]; !ok || modT.After(cur.mod) {
				latest[key] = newest{path: path, mod: modT}
			}
		}
	}

	for _, v := range latest {
		if _, err := sync.Sync(v.path); err != nil {
			log.Printf("initial sync error for %s: %v", v.path, err)
		}
	}
}

func buildFilesState(cfg config.Config) map[string]string {
	hashes := make(map[string]string)
	for _, agent := range cfg.Agents {
		for _, path := range agent.Files() {
			if h, err := util.FileHash(path); err != nil {
				log.Printf("hash error %s: %v", path, err)
			} else {
				hashes[path] = h
			}
		}
	}
	return hashes
}
