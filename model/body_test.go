package model

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFold(t *testing.T) {
	msg := strings.Join([]string{
		"Subject: Very important mail",
		"",
		"This is important!",
	}, "\r\n")

	expected := strings.Join([]string{
		"Received: by very.good.mail.server (briefmail) for",
		" <a-very-important-person@very.good.mail.server>; Sat, 5 Jan 2019 06:33:36",
		" +0000 (UTC)",
		"Subject: Very important mail",
		"",
		"This is important!",
	}, "\r\n")

	body := Body{Reader: strings.NewReader(msg)}
	body.Prepend(
		"Received",
		"by very.good.mail.server (briefmail) "+
			"for <a-very-important-person@very.good.mail.server>"+
			"; Sat, 5 Jan 2019 06:33:36 +0000 (UTC)")

	var actual bytes.Buffer
	actual.ReadFrom(body)

	assert.Equal(t, expected, actual.String())
}
