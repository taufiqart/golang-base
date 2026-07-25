package mailer

import (
	"crypto/tls"

	"gopkg.in/gomail.v2"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type GoMailer struct {
	dialer *gomail.Dialer
	from   string
}

func NewGoMailer(cfg Config) *GoMailer {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true} // adjust for production

	return &GoMailer{
		dialer: d,
		from:   cfg.From,
	}
}

func (m *GoMailer) Send(msg *Message) error {
	gm := gomail.NewMessage()
	gm.SetHeader("From", m.from)
	gm.SetHeader("To", msg.To...)
	if len(msg.CC) > 0 {
		gm.SetHeader("Cc", msg.CC...)
	}
	if len(msg.BCC) > 0 {
		gm.SetHeader("Bcc", msg.BCC...)
	}
	gm.SetHeader("Subject", msg.Subject)

	contentType := "text/plain"
	if msg.IsHTML {
		contentType = "text/html"
	}
	gm.SetBody(contentType, msg.Body)

	for _, attachment := range msg.Attachments {
		gm.Attach(attachment)
	}

	return m.dialer.DialAndSend(gm)
}
