package mailer

type Message struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	IsHTML      bool
	Attachments []string
}

type Mailer interface {
	Send(msg *Message) error
}
