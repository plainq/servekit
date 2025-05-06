package slackkit

import (
	"fmt"
)

// Error is a custom error type for Slack notifications.
type Error string

func (e Error) Error() string { return string(e) }

// Slack notification errors.
const (
	// ErrNilBlock is returned when a block is nil.
	ErrNilBlock Error = "nil block"

	// ErrNilText is returned when a text is nil.
	ErrNilText Error = "nil text"

	// ErrInvalidBlockType is returned when a block type is invalid.
	ErrInvalidBlockType Error = "invalid block type"

	// ErrInvalidBlockTextType is returned when a block text type is invalid.
	ErrInvalidBlockTextType Error = "invalid block text type"

	// ErrInvalidBlockTextStyle is returned when a block text style is invalid.
	ErrInvalidBlockTextStyle Error = "invalid block text style"

	// ErrInvalidBlockTextEmoji is returned when a block text emoji is invalid.
	ErrInvalidBlockTextEmoji Error = "invalid block text emoji"
)

// NewNotification creates a new notification.
func NewNotification(blocks ...Block) (*Notification, error) {
	for i, block := range blocks {
		if err := block.validate(); err != nil {
			return nil, fmt.Errorf("block %d is invalid: %w", i, err)
		}
	}

	return &Notification{Blocks: blocks}, nil
}

// Block types are defined in the Slack API documentation.
type BlockType string

// Block types are defined in the Slack API documentation.
const (
	Header  BlockType = "header"
	Divider BlockType = "divider"
	Section BlockType = "section"
)

// Text types are defined in the Slack API documentation.
type TextType string

const (
	// PlainText is the default text type.
	PlainText TextType = "plain_text"

	// Markdown is the markdown text type.
	Markdown TextType = "mrkdwn"
)

// Notification is the main struct for sending notifications to Slack.
type Notification struct {
	Blocks []Block `json:"blocks"`
}

// Block is a single block of a notification.
type Block struct {
	Type BlockType `json:"type"`
	Text *Text     `json:"text,omitempty"`
}

// validate checks if the block is valid.
func (b *Block) validate() error {
	if b == nil {
		return ErrNilBlock
	}

	switch b.Type {
	case Header:
		if b.Text == nil {
			return ErrNilText
		}

		if b.Text.Style != nil {
			return ErrInvalidBlockTextStyle
		}

		if b.Text.Type != PlainText {
			return ErrInvalidBlockTextType
		}

		if b.Text.Text == "" {
			return ErrNilText
		}

	default:
		return ErrInvalidBlockType
	}

	return nil
}

// Text is a single text block of a notification.
type Text struct {
	Type  TextType `json:"type"`
	Text  string   `json:"text"`
	Emoji *bool    `json:"emoji,omitempty"`
	Style *Style   `json:"style,omitempty"`
}

// Style is a style block of a notification.
type Style struct {
	Bold   *bool `json:"bold,omitempty"`
	Italic *bool `json:"italic,omitempty"`
	Strike *bool `json:"strike,omitempty"`
}

// NewHeader creates a new header block.
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

// NewDivider creates a new divider block.
func NewDivider() Block {
	return Block{
		Type: Divider,
	}
}

// NewSection creates a new section block.
func NewSection(text string, emoji bool) Block {
	return Block{
		Type: Section,
		Text: &Text{
			Type:  PlainText,
			Text:  text,
			Emoji: &emoji,
		},
	}
}
