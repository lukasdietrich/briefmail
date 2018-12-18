package model

import (
	"time"
)

type Envelope struct {
	Helo string
	Addr string
	Date time.Time
	From *Address
	To   []*Address
}
