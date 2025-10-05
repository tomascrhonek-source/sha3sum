package main

import (
	"crypto/sha3"
	"database/sql"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
	connStr := "user=" + dbUser + "dbname=" + dbName + "host=" + dbHost + "password='" + dbPassword + "' sslmode=disable"
	db, err := sql.Open("postgres", connStr)
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
}

func main() {
	logging := flag.Bool("debug", false, "Enable debug")
	timming := flag.Bool("timming", false, "Enable timming")
	root := flag.String("root", ".", "Root directory")

	flag.Parse()

	config()

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

	ch := make(chan entry, 10000000)

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
		if *logging {
			log.Println("Inserted:", e.path)
		}
	}

	if err != nil {
		log.Println("Error:", err)
	}

	timeEnd := time.Now()
	if *timming {
		log.Println("End:", timeEnd)
		log.Println("Duration:", timeEnd.Sub(timeStart))
		log.Println("sha3sum finished")
	}
}
