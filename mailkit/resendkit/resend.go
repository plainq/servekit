package resendkit

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/plainq/servekit/mailkit"
	"github.com/resend/resend-go/v2"
)

// ResendSender represents a type that is responsible for sending email messages using the Resend service.
type ResendSender struct {
	client *resend.Client
}

// Option is a type representing a function that modifies a ResendSender.
type Option func(*ResendSender)

// NewResendSender is a function that creates a new ResendSender instance.
func NewResendSender(apikey string, options ...Option) *ResendSender {
	s := ResendSender{
		client: resend.NewClient(apikey),
	}

	for _, option := range options {
		option(&s)
	}

	return &s
}

func (s *ResendSender) Send(ctx context.Context, message mailkit.Message) error {
	msgToSend := resend.SendEmailRequest{
		From:        message.From,
		To:          slices.Clone[[]string](message.To),
		Subject:     message.Subject,
		Bcc:         slices.Clone[[]string](message.Bcc),
		Cc:          slices.Clone[[]string](message.Cc),
		ReplyTo:     message.ReplyTo,
		Html:        message.HTML,
		Text:        message.Text,
		Tags:        make([]resend.Tag, 0, len(message.Tags)),
		Attachments: make([]*resend.Attachment, 0, len(message.Attachments)),
		Headers:     maps.Clone[map[string]string](message.Headers),
	}

	for _, attachment := range message.Attachments {
		resendAttachment := &resend.Attachment{
			Content:     attachment.Content,
			Filename:    attachment.Filename,
			Path:        attachment.Path,
			ContentType: attachment.ContentType,
		}

		msgToSend.Attachments = append(msgToSend.Attachments, resendAttachment)
	}

	for _, tag := range message.Tags {
		resendTag := resend.Tag{
			Name:  tag.Name,
			Value: tag.Value,
		}

		msgToSend.Tags = append(msgToSend.Tags, resendTag)
	}

	if _, err := s.client.Emails.SendWithContext(ctx, &msgToSend); err != nil {
		return fmt.Errorf("resend: sending email: %w", err)
	}

	return nil
}
