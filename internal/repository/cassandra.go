package repository

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	_ "github.com/lib/pq"

	"sync"
	"therealbroker/pkg/broker"
	"time"
)

var (
	cassandraDB *CassandraDatabase
)

const (
	CASS_HOST     = "localhost"
	CASS_PORT     = "9042"
	CASS_USER     = "cassandra"
	CASS_PASSWORD = "cassandra"
	CASS_NAME     = "broker"
)

type CassandraDatabase struct {
	sync.Mutex
	client         *gocql.Session
	deleteMessages []string
	lastID         int64
}

func (db *CassandraDatabase) createTable() error {
	err := db.client.Query(`CREATE TABLE IF NOT EXISTS broker.messages (id int, subject text, body text, expiration_date bigint, PRIMARY KEY (id));`).Exec()
	if err != nil {
		log.Println(err)

		return err
	}

	var exists string
	_ = db.client.Query(`SELECT table_name FROM system_schema.tables WHERE keyspace_name='broker';`).Iter().Scan(&exists)

	err = db.client.Query(`CREATE TABLE IF NOT EXISTS broker.ids (id_name varchar, next_id int, PRIMARY KEY (id_name));`).Exec()
	if err != nil {
		log.Println(err)

		return err
	}

	if exists != "ids" {
		err = db.client.Query(`INSERT INTO broker.ids (id_name, next_id) VALUES ('messages_id', 1);`).Exec()
		if err != nil {
			log.Println(err)

			return err
		}

		db.lastID = 1
	} else {
		var lastID int
		ok := db.client.Query(`SELECT next_id FROM broker.ids WHERE id_name='messages_id';`).Iter().Scan(&lastID)
		if !ok {
			log.Println("error getting last id")

			return fmt.Errorf("error getting last id")
		}
		db.lastID = int64(lastID)
	}

	log.Println(db.lastID)

	return nil
}

func (db *CassandraDatabase) SaveMessage(msg broker.Message, subject string) int {
	db.lastID++

	query := fmt.Sprintf(`INSERT INTO broker.messages (id, subject, body, expiration_date) VALUES (%d, '%s', '%s', %v);`, db.lastID, subject, msg.Body, int64(msg.Expiration))

	err := db.client.Query(query).Exec()
	if err != nil {
		fmt.Println("saving error:", err)
		return -1
	}

	query = fmt.Sprintf(`UPDATE broker.ids SET next_id=%d WHERE id_name='messages_id';`, db.lastID)

	err = db.client.Query(query).Exec()
	if err != nil {
		fmt.Println("saving error:", err)
		return -1
	}

	return int(db.lastID)
}

func (db *CassandraDatabase) FetchMessage(id int) (broker.Message, error) {
	query := fmt.Sprintf("SELECT body, expiration_date from broker.messages where id=%d;", id)

	rows := db.client.Query(query).Iter()
	// if err != nil {
	// 	fmt.Println("fetch: returned from query")
	// 	return broker.Message{}, err
	// }

	var body string

	var expirationDate int64

	ok := rows.Scan(&body, &expirationDate)
	if !ok {
		fmt.Println("fetch: scan error")
		return broker.Message{}, fmt.Errorf("scan error")
	}

	if body == "" {
		return broker.Message{}, fmt.Errorf("message not found")
	}

	msg := broker.Message{
		Body:       body,
		Expiration: time.Duration(expirationDate),
	}
	rows.Close()

	return msg, nil
}

func (db *CassandraDatabase) DeleteMessage(id int, subject string) {
	db.Lock()
	db.deleteMessages = append(db.deleteMessages, fmt.Sprintf("%d", id))
	db.Unlock()
}

func (db *CassandraDatabase) batchHandler(ticker *time.Ticker) {
	for range ticker.C {
		db.Lock()
		if len(db.deleteMessages) != 0 {
			query := `DELETE FROM broker.messages WHERE id IN (` + strings.Join(db.deleteMessages, " , ") + ");"
			db.deleteMessages = db.deleteMessages[:0]
			log.Println(query)

			err := db.client.Query(query).Exec()
			if err != nil {
				fmt.Println(err)
			}
		}
		db.Unlock()
	}
}

func GetCassandra() (Database, error) {
	var once sync.Once

	once.Do(func() {
		cluster := gocql.NewCluster(CASS_HOST)
		cluster.Port, _ = strconv.Atoi(CASS_PORT)
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: CASS_USER,
			Password: CASS_PASSWORD,
		}
		cluster.Consistency = gocql.Quorum
		cluster.Timeout = time.Second * 1000
		session, err := cluster.CreateSession()
		if err != nil {
			log.Fatal(err)
		}

		if err := session.Query(`CREATE KEYSPACE IF NOT EXISTS broker WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };`).Exec(); err != nil {
			log.Fatal(err)
		}

		cassandraDB = &CassandraDatabase{
			client:         session,
			deleteMessages: make([]string, 0),
		}

		err = cassandraDB.createTable()
		if err != nil {
			connectionError = err

			return
		}

		// err = cassandraDB.client.Query(`SELECT id FROM broker.messages WITH CLUSTERING ORDER BY (lastUpdated DESC);`).Scan(&lastID)
		// if err != nil {
		// log.Fatal(err)
		// }

		ticker := time.NewTicker(1 * time.Second)

		go cassandraDB.batchHandler(ticker)
	})

	return cassandraDB, connectionError
}
