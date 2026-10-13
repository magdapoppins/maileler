package gcloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// SendEmail sends a plain-text email from the authenticated user to the given address.
func (c *Client) SendEmail(ctx context.Context, to, subject, body string) error {
	raw, err := buildMessage(to, subject, body)
	if err != nil {
		return err
	}

	// "me" refers to the user the token was issued for.
	_, err = c.gmail.Users.Messages.Send("me", &gmail.Message{Raw: raw}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// buildMessage builds an RFC 2822 message and encodes it the way the Gmail API expects.
func buildMessage(to, subject, body string) (string, error) {
	addr, err := mail.ParseAddress(to)
	if err != nil {
		return "", fmt.Errorf("invalid recipient %q: %w", to, err)
	}
	// Newlines in the subject would let it inject extra headers.
	if strings.ContainsAny(subject, "\r\n") {
		return "", fmt.Errorf("subject must not contain newlines")
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "To: %s\r\n", addr.String())
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(base64.StdEncoding.EncodeToString([]byte(body)))

	return base64.URLEncoding.EncodeToString([]byte(msg.String())), nil
}
