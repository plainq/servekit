package slackkit

import (
	"fmt"
)

func NewNotification(blocks ...Block) (*Notification, error) {
	for i, block := range blocks {
		if err := block.validate(); err != nil {
			return nil, fmt.Errorf("block %d is invalid: %w", i, err)
		}
	}

	return &Notification{Blocks: blocks}, nil
}

type BlockType string

const (
	Header  BlockType = "header"
	Divider BlockType = "divider"
	Section BlockType = "section"
)

type TextType string

const (
	PlainText TextType = "plain_text"
	Markdown  TextType = "mrkdwn"
)

type Notification struct {
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Type BlockType `json:"type"`
	Text *Text     `json:"text,omitempty"`
}

func (b *Block) validate() error {
	if b == nil {
		return fmt.Errorf("nil block")
	}

	switch b.Type {
	case Header:
		if b.Text == nil {
			return fmt.Errorf("header block does not have a text block")
		}

		if b.Text.Style != nil {
			return fmt.Errorf("header block should not contain style block")
		}

		if b.Text.Type != PlainText {
			return fmt.Errorf("header text block type should be plain_text only")
		}

		if b.Text.Text == "" {
			return fmt.Errorf("header text block does not contain any text")
		}

	default:
		return fmt.Errorf("unknown block type")
	}

	return nil
}

type Text struct {
	Type  TextType `json:"type"`
	Text  string   `json:"text"`
	Emoji *bool    `json:"emoji,omitempty"`
	Style *Style   `json:"style,omitempty"`
}

type Style struct {
	Bold   *bool `json:"bold,omitempty"`
	Italic *bool `json:"italic,omitempty"`
	Strike *bool `json:"strike,omitempty"`
}

func NewHeader(text string, emoji bool) Block {
	return Block{
		Type: Header,
		Text: &Text{
			Type:  PlainText,
			Text:  text,
			Emoji: &emoji,
		},
	}
}
