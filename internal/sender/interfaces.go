package sender

import "context"

type Message struct {
	To      string
	Subject string
	Body    string
	HTML    string
}

type EmailSender interface {
	Send(ctx context.Context, msg Message) error
}
