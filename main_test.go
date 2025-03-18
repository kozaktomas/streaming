package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJsonFromResponseWithJson(t *testing.T) {
	input := `
I apologize for the oversight. Let’s ensure the JSON is formatted correctly without special characters that might break the parsing:

\\\json
[
	"1: Bootování systému vašeho ega...",
	"2: Kontrola kávy ve stylu DevOps...",
	"3: Aktivace únikového klíče ESC...",
	"4: Nabíhání výrazu \"tohle by si mělo samo updatovat\"...",
	"5: Příprava Google na odpovědi..."
]

Make sure you copy it correctly into your JSON parser or environment. This array of strings does not have special JSON object structure issues, as it’s a simplified version. Let me know if this resolves the error!
`
	items, err := parseJsonFromResponse(input)
	assert.NoError(t, err)
	assert.Len(t, items, 5)
}

func TestParseJsonFromResponseNoJson(t *testing.T) {
	input := `This is random string without json`
	_, err := parseJsonFromResponse(input)
	assert.Error(t, err)
}
