package repository

import "therealbroker/pkg/broker"

type Database interface {
	SaveMessage(msg broker.Message, subject string) int
	FetchMessage(id int) (broker.Message, error)
	DeleteMessage(id int, subject string)
}
