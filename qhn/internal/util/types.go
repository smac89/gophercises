package util

import (
	"gophercises.com/qhn/hn"
	"time"
)

// Item is the same as the hn.Item, but adds the Host field
type Item struct {
	hn.Item
	Host string
}

type TemplateData struct {
	Stories []Item
	Time    time.Duration
}

type Range struct {
	Start, End uint
}
