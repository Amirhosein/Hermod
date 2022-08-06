package model

import "therealbroker/pkg/broker"

type Subscriber struct {
	Channel chan broker.Message
}
