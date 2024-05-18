package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var TemplateFS embed.FS

const (
	timeout = 5 * time.Second
)

type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host, username, password, sender string, port int) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = timeout

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

func (m Mailer) Send(recipient, tf string, data any) error {
	t, err := template.New("email").ParseFS(TemplateFS, "templates/"+tf)
	if err != nil {
		slog.Info("i am the problem, 38")
		return err
	}

	subject := new(bytes.Buffer)
	if err := t.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}

	body := new(bytes.Buffer)
	if err := t.ExecuteTemplate(body, "body", data); err != nil {
		return err
	}

	html := new(bytes.Buffer)
	if err := t.ExecuteTemplate(html, "html", data); err != nil {
		return err
	}

	msg := mail.NewMessage()

	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())

	msg.SetBody("text/plain", body.String())
	msg.AddAlternative("text/html", html.String())

	if err := m.dialer.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
