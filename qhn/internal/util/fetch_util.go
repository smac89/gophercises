package util

import (
	"gophercises.com/qhn/hn"
	"net/url"
	"strings"
)

func FetchItemChan(ch chan<- *hn.Item, client *hn.Client, id int) {
	hnItem, err := client.GetItem(id)
	if err == nil {
		ch <- &hnItem
	} else {
		ch <- nil
	}
}

func IsStoryLink(item Item) bool {
	return item.Type == "story" && item.URL != ""
}

func ParseHNItem(hnItem hn.Item) Item {
	ret := Item{Item: hnItem}
	u, err := url.Parse(ret.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(u.Hostname(), "www.")
	}
	return ret
}
