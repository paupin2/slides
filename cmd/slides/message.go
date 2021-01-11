package main

type Message struct {
	Show  string `json:"show,omitempty"`
	Cksum uint32 `json:"cksum,omitempty"`
	Size  int    `json:"size,omitempty"`
	Text  string `json:"text,omitempty"`
}

const (
	MaxSlideCount = 1000
)

func (m Message) Valid() bool {
	if m.Show != "" && m.Text != "" {
		// both set
		return false
	}
	if m.Size < 0 {
		// bad size of size without show
		return false
	}
	return true
}
