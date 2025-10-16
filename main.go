package main

import (
	"crypto/sha3"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type entry struct {
	path string
	hash []byte
	size int64
}

var pool sync.Map

func init() {
	pool = sync.Map{}
}

func main() {
	cmdLogging := flag.Bool("debug", false, "Enable debug")
	cmdTimming := flag.Bool("timming", false, "Enable timming")
	cmdNoDB := flag.Bool("nodb", false, "Disable connection to the database")
	cmdRoot := flag.String("root", ".", "Root directory")

	flag.Parse()

	cfg := configure()

	if *cmdLogging == true {
		cfg.logging = cmdLogging
	}
	if *cmdTimming == true {
		cfg.timming = cmdTimming
	}
	if *cmdRoot != "." {
		cfg.root = cmdRoot
	}
	if *cmdNoDB == true {
		cfg.nodb = cmdNoDB
	}

	if *cfg.logging {
		log.Println("Debug enabled")
		log.Println("Root directory:", *cfg.root)
	}

	if *cfg.logging && *cfg.nodb {
		log.Println("No database connection enabled")
	}

	timeStart := time.Now()
	if *cfg.timming {
		log.Println("Timming enabled")
		log.Println("Start:", timeStart)
	}

	if *cfg.logging {
		log.SetOutput(os.Stderr)
		log.Println("sha3sum started")
	}

	if *cfg.logging {
		log.Println("Start:", timeStart)
	}

	var db *sql.DB

	if !*cfg.nodb {
		db = dbConnect(cfg)
		defer db.Close()
	}

	walkDirectoryTree(cfg)
	if !*cfg.nodb {
		saveToDB(db, cfg.logging)
	}

	timeEnd := time.Now()
	if *cfg.timming {
		log.Println("End:", timeEnd)
		log.Println("Duration:", timeEnd.Sub(timeStart))
		log.Println("sha3sum finished")
	}
}

func walkDirectoryTree(cfg config) {
	var wg sync.WaitGroup
	wg = sync.WaitGroup{}

	err := filepath.Walk(*cfg.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if *cfg.logging {
			log.Println(path)
		}

		fi, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if fi.Mode().IsRegular() {
			wg.Go(func() {
				entry := computeHash(path)
				if *cfg.nodb {
					fmt.Println(hex.EncodeToString(entry.hash), entry.path)
				} else {
					pool.Store(entry.path, entry)
				}
			})
		}

		return nil
	})
	if err != nil {
		log.Println("Error:", err)
	}
}

func computeHash(path string) entry {
	f, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	hash := sha3.New512()

	written, err := io.Copy(hash, f)
	if err != nil {
		log.Fatal(err)
	}

	bs := hash.Sum(nil)

	return entry{path: path, hash: bs, size: written}
}
