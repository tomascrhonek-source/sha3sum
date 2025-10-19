# sha3sum
Tool for summing files using the sha3 standard. Like the unix standard shasum and sha2sum.

sha3sum uses PostgreSQL database for store the sums of the files. This is important for summing files accros the network.

It uses threading using sync.Map and sync.WaitGroup to compute the sha3 sum per file. Count of threads is limited by Golang runtime and can be adjusted by runtime.GOMAXPROCS(2) in the code.

Together with the package, a systemd service sha3sum.service and sha3sum.timer are installed. Adjust the regular execution time and activate the timer if you want.

The database config is in file ~/.config/sha3sum.yaml od /etc/sha3sum.yaml. You can change it in the code (config.go). Feel free to do.

    database:
        dbname: dbname
        host: localhost
        password: changeme
        port: 5432
        user: dbuser
    config:
        root: "."
        debug: false
        timming: false
        nodb: false

## Usage

sha3sum -debug -timming -nodb -root dir

Flags arguments have priority prior the config.yaml file.

Where:
- root - the root directory of the tree with files
- debug - print debug messages (a lot)
- timming - prints times during the operations, usefull for benchmarking
- nodb - print the sha3 sums on stdout and do not connect to the database
