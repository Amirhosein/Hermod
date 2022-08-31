package repository

import "therealbroker/internal/pkg/broker"

type Database interface {
	SaveMessage(msg broker.Message, subject string) int
	FetchMessage(id int) (broker.Message, error)
	DeleteMessage(id int, subject string)
}
