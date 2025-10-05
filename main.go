package main

import (
	"crypto/sha3"
	"database/sql"
	"encoding/hex"
	"flag"
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
	if int64(n) != size {
		log.Fatalf("Read %d bytes, expected %d bytes", n, size)
	}

	hash := sha3.New512()
	hash.Write(buffer)
	bs := hash.Sum(nil)

	ch <- entry{path: path, hash: bs, size: size}
}

func dbConnect() *sql.DB {
	connStr := "user=tomas dbname=tomas host=185.156.37.17 password='gKSGetXshQbd69Qte85LROSG' sslmode=disable"
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

var dbName string
var dbUser string
var dbPassword string
var dbHost string
var dbPort int
var logging *bool
var timming *bool
var root *string

func config() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

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
		viper.Set("threading", true)

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

	ch := make(chan entry, runtime.NumCPU())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go walkDirectoryTree(root, logging, ch)
	wg.Done()
	saveToDB(db, ch, logging)
	wg.Wait()

	timeEnd := time.Now()
	if *timming {
		log.Println("End:", timeEnd)
		log.Println("Duration:", timeEnd.Sub(timeStart))
		log.Println("sha3sum finished")
	}
}

func walkDirectoryTree(root *string, logging *bool, ch chan entry) {
	wg := sync.WaitGroup{}
	cpus := runtime.NumCPU()
	if viper.GetBool("config.threading") {
		runtime.GOMAXPROCS(cpus)
		if *logging {
			log.Println("Threading enabled, using", cpus, "CPUs")
		}
	}

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
			func() {
				if viper.GetBool("config.threading") {
					wg.Add(1)
				}
				computeHash(path, ch)
				if viper.GetBool("config.threading") {
					wg.Done()
				}
			}()
		}

		return nil
	})
	if err != nil {
		log.Println("Error:", err)
	}

	wg.Wait()
	close(ch)
}

func saveToDB(db *sql.DB, ch chan entry, logging *bool) {
	if *logging {
		log.Println("Saving to database")
	}

	for range ch {
		e := <-ch
		_, err := db.Exec("INSERT INTO sha3sum (path, sum, size) VALUES ($1, $2, $3)", e.path, hex.EncodeToString(e.hash), e.size)
		if err != nil {
			log.Println("Error:", err)
		}
		if *logging {
			log.Println("Inserted:", e.path)
		}
	}

	if *logging {
		log.Println("All entries inserted")
	}
}
