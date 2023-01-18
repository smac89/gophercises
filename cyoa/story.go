package cyoa

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

type Story map[string]Chapter

type Option struct {
	Text    string `json:"text"`
	Chapter string `json:"arc"`
}

type Chapter struct {
	Title      string   `json:"title"`
	Paragraphs []string `json:"story"`
	Options    []Option `json:"options"`
}

func (story *Story) DecodeJson(r io.Reader) error {
	d := json.NewDecoder(r)
	if err := d.Decode(story); err != nil {
		return fmt.Errorf("failed to decode json story. %v", err)
	}
	return nil
}
