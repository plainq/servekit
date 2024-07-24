package mailkit

import (
	"context"
)

// Sender holds logic of sending an email messages.
type Sender interface {
	// Send sends an email message.
	Send(ctx context.Context, message Message) error
}

// Email represents an email address.
// Holds name and address.
type Email struct {
	Name    string
	Address string
}

// Message represents email message.
type Message struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	Bcc         []string          `json:"bcc,omitempty"`
	Cc          []string          `json:"cc,omitempty"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	Tags        []Tag             `json:"tags,omitempty"`
	Attachments []*Attachment     `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Tag is used to define custom metadata for message.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment is the public struct used for adding attachments to emails.
type Attachment struct {
	// Content is the binary content of the attachment to use when a Path
	// is not available.
	Content []byte `json:"content"`

	// Filename that will appear in the email.
	// Make sure you pick the correct extension otherwise preview
	// may not work as expected.
	Filename string `json:"filename"`

	// Path where the attachment file is hosted instead of providing the
	// content directly.
	Path string `json:"path"`

	// Content type for the attachment, if not set will be derived from
	// the filename property.
	ContentType string `json:"contentType"`
}
