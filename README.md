# sha3sum
Program for summing files using the sha3 standard. Like the unix standard shasum and sha2sum.

sha3sum uses PostgreSQL database for store the sums of the files. This is important for summing files accros the network.

The database config is in file confing.yaml in root of the repository. You can change it in the code. Feel free to do.

database:
    dbname: dbname
    host: localhost
    password: changeme
    port: 5432
    user: dbuser

## Usage

sha3sum -debug -timming -root dir

Where:
- root - the root directory of the tree with files
- debug - print debug messages (a lot)
- timming - prints times during the operations, usefull for benchmarking

