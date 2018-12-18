package model

import (
	"errors"
	"strings"
)

var (
	ErrInvalidAddressFormat = errors.New("address: invalid format")
)

type Address struct {
	User   string
	Domain string
}

func ParseAddress(raw string) (*Address, error) {
	if i := strings.LastIndex(raw, "@"); i > -1 {
		return &Address{
			User:   raw[:i],
			Domain: raw[i+1:],
		}, nil
	}

	return nil, ErrInvalidAddressFormat
}

func (a *Address) String() string {
	return a.User + "@" + a.Domain
}
