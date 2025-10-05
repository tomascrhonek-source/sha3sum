package main

import (
	"crypto/sha3"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

type entry struct {
	path string
	hash []byte
	size int64
}

const logging = false
const timming = true

func computeHash(path string, ch chan entry) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	size := fi.Size()
	buffer := make([]byte, size)
	n, err := f.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	if logging {
		log.Println("File size:", n)
	}
	hash := sha3.New512()
	hash.Write(buffer)
	bs := hash.Sum(nil)
	if logging {
		log.Printf("SHA3-512: %x\n", bs)
	}

	ch <- entry{path: path, hash: bs, size: size}
}

func dbConnect() *sql.DB {
	connStr := "user=tomas dbname=tomas host=192.168.42.188 password='bk9qqmzB6E16dBGTS6gkvTrX' sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func main() {
	timeStart := time.Now()
	if timming {
		log.Println("Timming enabled")
		log.Println("Start:", timeStart)
	}

	if logging {
		log.SetOutput(os.Stderr)
		log.Println("sha3sum started")
	}

	root := "d:\\Gay\\"

	if logging {
		log.Println("Start:", timeStart)
	}

	db := dbConnect()
	defer db.Close()

	ch := make(chan entry, 10000000)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if logging {
			log.Println(path)
		}

		fi, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if fi.IsDir() == false {
			computeHash(path, ch)
		}

		return nil
	})
	if err != nil {
		log.Println("Error:", err)
	}

	close(ch)

	for range ch {
		e := <-ch
		_, err = db.Exec("INSERT INTO sha3sum (path, sum, size) VALUES ($1, $2, $3)", e.path, hex.EncodeToString(e.hash), e.size)
		if err != nil {
			log.Println("Error:", err)
		}
		if logging {
			log.Println("Inserted:", e.path)
		}
	}

	if err != nil {
		log.Println("Error:", err)
	}

	timeEnd := time.Now()
	if timming {
		log.Println("End:", timeEnd)
		log.Println("Duration:", timeEnd.Sub(timeStart))
		log.Println("sha3sum finished")
	}
}
