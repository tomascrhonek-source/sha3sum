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

type config struct {
	dbName         string
	dbUser         string
	dbPassword     string
	dbHost         string
	dbPort         int
	logging        *bool
	timming        *bool
	nodb           *bool
	maxConnections int
	root           *string
}

var pool sync.Map

func init() {
	pool = sync.Map{}
}

func configure() config {
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
		viper.Set("config.maxconnections", 10)
		viper.Set("config.nodb", false)

		viper.SafeWriteConfig()
	}

	cfg := config{}
	cfg.dbHost = viper.GetString("database.host")
	cfg.dbPort, err = strconv.Atoi(viper.GetString("database.port"))
	if err != nil {
		cfg.dbPort = 5432
	}
	cfg.dbUser = viper.GetString("database.user")
	cfg.dbPassword = viper.GetString("database.password")
	cfg.dbName = viper.GetString("database.dbname")

	cfg.logging = new(bool)
	cfg.timming = new(bool)
	cfg.nodb = new(bool)
	cfg.root = new(string)
	cfg.maxConnections, err = strconv.Atoi(viper.GetString("config.maxconnections"))
	if err != nil {
		cfg.maxConnections = 10
	}
	*cfg.logging = viper.GetBool("config.debug")
	*cfg.timming = viper.GetBool("config.timming")
	*cfg.nodb = viper.GetBool("config.nodb")
	*cfg.root = viper.GetString("config.root")

	return cfg
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

	db := dbConnect(cfg)
	defer db.Close()

	walkDirectoryTree(cfg)
	saveToDB(db, cfg.logging)

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

func saveToDB(db *sql.DB, logging *bool) {
	pool.Range(func(key any, value any) bool {
		_, err := db.Exec("INSERT INTO sha3sum (path, sum, size) VALUES ($1, $2, $3)", value.(entry).path, hex.EncodeToString(value.(entry).hash), value.(entry).size)
		if err != nil {
			log.Println("Error:", err)
		}
		if *logging {
			log.Println("Inserted:", value.(entry).path)
		}

		if *logging {
			log.Println("All entries inserted")
		}
		return true
	})
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

func dbConnect(cfg config) *sql.DB {
	connStr := fmt.Sprintf("user=%s dbname=%s host=%s password='%s' port=%d sslmode=disable",
		cfg.dbUser, cfg.dbName, cfg.dbHost, cfg.dbPassword, cfg.dbPort)
	if *cfg.logging {
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
