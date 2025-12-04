package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//Ember the email templates

//go:embed templates/*
var templateFS embed.FS

// Mailer wraps the SMTP dialer and sender info
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

// New returns a Mailer configured with SMTP settings
func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}

}

// Send composes and sends an email using the given template and data
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	//Load templates form embedded FS
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	//Render subject, plain text, and HTML bodies
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	//Build the email message
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Send the message.
	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	return nil

}
