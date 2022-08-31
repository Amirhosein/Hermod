package model

import "therealbroker/internal/pkg/broker"

type Subscriber struct {
	Channel chan broker.Message
}
