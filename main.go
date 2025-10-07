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
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"

	_ "github.com/lib/pq"
)

type entry struct {
	path string
	hash []byte
	size int64
}

var dbName string
var dbUser string
var dbPassword string
var dbHost string
var dbPort int
var logging *bool
var timming *bool
var threading *bool
var root *string

func config() {
	viper.SetConfigName("sha3sum")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/")

	err := viper.ReadInConfig()
	if err != nil {
		viper.Set("database.host", "localhost")
		viper.Set("database.port", 5432)
		viper.Set("database.user", "dbuser")
		viper.Set("database.password", "dbpassword")
		viper.Set("database.dbname", "dbname")

		viper.Set("config.debug", false)
		viper.Set("config.timming", false)
		viper.Set("config.root", ".")
		viper.Set("config.threading", true)

		viper.SafeWriteConfig()
	}

	dbHost = viper.GetString("database.host")
	dbPort, err = strconv.Atoi(viper.GetString("database.port"))
	if err != nil {
		dbPort = 5432
	}
	dbUser = viper.GetString("database.user")
	dbPassword = viper.GetString("database.password")
	dbName = viper.GetString("database.dbname")

	logging = new(bool)
	timming = new(bool)
	root = new(string)
	*logging = viper.GetBool("config.debug")
	*timming = viper.GetBool("config.timming")
	*root = viper.GetString("config.root")
}

func main() {
	cmdLogging := flag.Bool("debug", false, "Enable debug")
	cmdTimming := flag.Bool("timming", false, "Enable timming")
	cmdThreading := flag.Bool("threading", false, "Enable threading")
	cmdRoot := flag.String("root", ".", "Root directory")

	flag.Parse()

	config()

	if *cmdLogging == true {
		logging = cmdLogging
	}
	if *cmdTimming == true {
		timming = cmdTimming
	}
	if *cmdRoot != "." {
		root = cmdRoot
	}
	if *cmdThreading == true {
		threading = cmdThreading
	}

	if *logging {
		log.Println("Debug enabled")
		log.Println("Root directory:", *root)
	}

	timeStart := time.Now()
	if *timming {
		log.Println("Timming enabled")
		log.Println("Start:", timeStart)
	}

	if *logging {
		log.SetOutput(os.Stderr)
		log.Println("sha3sum started")
	}

	if *logging {
		log.Println("Start:", timeStart)
	}

	db := dbConnect()
	defer db.Close()

	walkDirectoryTree(root, logging, db)

	timeEnd := time.Now()
	if *timming {
		log.Println("End:", timeEnd)
		log.Println("Duration:", timeEnd.Sub(timeStart))
		log.Println("sha3sum finished")
	}
}

func walkDirectoryTree(root *string, logging *bool, db *sql.DB) {
	if viper.GetBool("config.threading") {
		if *logging {
			log.Println("Threading enabled, using", runtime.NumCPU(), "CPUs")
		}
	}

	wg := sync.WaitGroup{}

	err := filepath.Walk(*root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if *logging {
			log.Println(path)
		}

		fi, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if fi.IsDir() == false {
			if viper.GetBool("config.threading") {
				wg.Go(func() {
					entry := computeHash(path)
					saveToDB(db, entry, logging)
				})
			} else {
				entry := computeHash(path)
				saveToDB(db, entry, logging)
			}
		}

		return nil
	})
	if err != nil {
		log.Println("Error:", err)
	}
	wg.Wait()
}

func saveToDB(db *sql.DB, entry entry, logging *bool) {
	_, err := db.Exec("INSERT INTO sha3sum (path, sum, size) VALUES ($1, $2, $3)", entry.path, hex.EncodeToString(entry.hash), entry.size)
	if err != nil {
		log.Println("Error:", err)
	}
	if *logging {
		log.Println("Inserted:", entry.path)
	}

	if *logging {
		log.Println("All entries inserted")
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

func dbConnect() *sql.DB {
	connStr := fmt.Sprintf("user=%s dbname=%s host=%s password='%s' port=%d sslmode=disable",
		dbUser, dbName, dbHost, dbPassword, dbPort)
	if *logging {
		log.Println("Connecting to database:", connStr)
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}
