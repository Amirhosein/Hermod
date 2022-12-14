package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
	_ "github.com/lib/pq"

	"sync"
	"therealbroker/internal/pkg/broker"
	"time"
)

var postgresDB *PostgresDatabase
var connectionError error

const (
	PG_HOST     = "postgres"
	PG_PORT     = "5432"
	PG_USER     = "postgres"
	PG_PASSWORD = "postgres"
	PG_NAME     = "broker"

	REDIS_HOST     = "redis"
	REDIS_PORT     = "6379"
	REDIS_PASSWORD = ""
)

type PostgresDatabase struct {
	sync.Mutex
	client         *sql.DB
	rClient        *redis.Client
	deleteMessages []string
	insertMessages []broker.Message
	lastID         int
}

func (db *PostgresDatabase) createTable() error {
	table := `
	CREATE TABLE IF NOT EXISTS messages (
		id serial,
		subject varchar(255) not null,
		body varchar(255) ,
		expiration_date bigint not null,
		primary key(id, subject)
	);`

	_, err := db.client.Exec(table)
	if err != nil {
		return err
	}

	return nil
}

func (db *PostgresDatabase) createIndex() error {
	command := `CREATE INDEX IF NOT EXISTS idx_id_subject on messages (id,subject)`

	_, err := db.client.Exec(command)
	if err != nil {
		return err
	}

	return nil
}

func (db *PostgresDatabase) SaveMessage(msg broker.Message, subject string) int {
	// query := fmt.Sprintf(`INSERT INTO messages(id, subject, body, expiration_date) VALUES (DEFAULT, '%s', '%s', %v) RETURNING id;`, subject, msg.Body, int64(msg.Expiration))

	db.lastID++
	msg.Id = db.lastID

	db.Lock()
	db.insertMessages = append(db.insertMessages, msg)
	db.Unlock()

	if len(db.insertMessages) > 10000 {
		db.batchInsert()
	}

	// var insertedID int

	// row, err := db.client.Query(query)
	// if err != nil {
	// 	fmt.Println("saving error:", err)
	// 	return -1
	// }

	// row.Next()
	// _ = row.Scan(&insertedID)
	// row.Close()

	return msg.Id
}

func (db *PostgresDatabase) FetchMessage(id int) (broker.Message, error) {
	query := fmt.Sprintf("SELECT body, expiration_date from messages where messages.id=%d;",
		id)

	rows, err := db.client.Query(query)
	if err != nil {
		fmt.Println("fetch: returned from query")
		return broker.Message{}, err
	}

	var body string

	var expirationDate int64

	for rows.Next() {
		err = rows.Scan(&body, &expirationDate)
		if err != nil {
			fmt.Println("fetch: scan error")
			return broker.Message{}, err
		}
	}

	if body == "" {
		return broker.Message{}, fmt.Errorf("message not found")
	}

	if err := rows.Err(); err != nil {
		fmt.Println("rows err: ", err)
	}

	msg := broker.Message{
		Body:       body,
		Expiration: time.Duration(expirationDate),
	}
	//rows.Close()

	return msg, nil
}

func (db *PostgresDatabase) DeleteMessage(id int, subject string) {
	db.Lock()
	db.deleteMessages = append(db.deleteMessages, fmt.Sprintf("(id,subject)=(%d,'%s')", id, subject))
	db.Unlock()
}

func (db *PostgresDatabase) batchHandler(ticker *time.Ticker) {
	for range ticker.C {
		db.Lock()
		if len(db.deleteMessages) != 0 {
			query := `DELETE FROM messages WHERE ` + strings.Join(db.deleteMessages, " or ") + ";"
			db.deleteMessages = db.deleteMessages[:0]

			_, err := db.client.Exec(query)
			if err != nil {
				fmt.Println(err)
			}
		}
		db.Unlock()

		db.batchInsert()
	}
}

func (db *PostgresDatabase) batchInsert() {
	if len(db.insertMessages) != 0 {
		query := `INSERT INTO messages(id, subject, body, expiration_date) VALUES `
		for _, msg := range db.insertMessages {
			query += fmt.Sprintf("(%d, '%s', '%s', %v),", msg.Id, "msg.Subject", msg.Body, int64(msg.Expiration))
			log.Println(msg.Id)
		}
		query = query[:len(query)-1] + ";"
		db.insertMessages = db.insertMessages[:0]

		db.rClient.Set("lastID", db.lastID, 0)
		log.Println("lastID:", db.lastID)

		_, err := db.client.Exec(query)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func GetPostgre() (Database, error) {
	time.Sleep(5 * time.Second)
	var once sync.Once

	once.Do(func() {
		connString := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable",
			PG_HOST, PG_PORT, PG_USER, PG_PASSWORD)

		client, err := sql.Open("postgres", connString)
		if err != nil {
			connectionError = err
			return
		}
		// defer client.Close()

		err = client.Ping()
		if err != nil {
			_, err = client.Exec("create database " + PG_NAME)
			if err != nil {
				log.Fatal(err)
			}
		}

		client.SetMaxOpenConns(90)
		client.SetMaxIdleConns(45)
		client.SetConnMaxIdleTime(time.Second * 10)
		postgresDB = &PostgresDatabase{
			client:         client,
			deleteMessages: make([]string, 0),
		}

		err = postgresDB.createTable()
		if err != nil {
			connectionError = err

			return
		}

		err = postgresDB.createIndex()
		if err != nil {
			connectionError = err

			return
		}

		rClient := redis.NewClient(&redis.Options{
			Addr:     REDIS_HOST + ":" + REDIS_PORT,
			Password: REDIS_PASSWORD,
		})

		_, err = rClient.Ping().Result()
		if err != nil {
			log.Fatal(err)
		} else {
			log.Println("redis connected")
		}

		postgresDB.rClient = rClient

		result, err := postgresDB.rClient.Get("lastID").Result()
		if err == redis.Nil {
			postgresDB.lastID = 600000
		} else if err != nil {
			log.Fatal(err)
		} else {
			postgresDB.lastID, _ = strconv.Atoi(result)
		}

		ticker := time.NewTicker(100 * time.Millisecond)

		go postgresDB.batchHandler(ticker)
	})

	return postgresDB, connectionError
}
