package notification

import (
	"net/smtp"
)

type EmailConfig struct {
	From     string
	Password string
	SMTPHost string
	SMTPPort string
}

func SendEmail(to string, subject string, body string, config EmailConfig) error {
	auth := smtp.PlainAuth("", config.From, config.Password, config.SMTPHost)

	message := []byte("Subject: " + subject + "\r\n\r\n" + body)

	err := smtp.SendMail(config.SMTPHost+":"+config.SMTPPort, auth, config.From, []string{to}, message)
	return err
}
