package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"go.uber.org/zap"
	gomail "gopkg.in/mail.v2"
)

type MailTrapClient struct {
	FromEmail       string
	smtpAddr        string
	smtpSandboxAddr string
	smtpPort        int
	username        string
	password        string
	logger          *zap.SugaredLogger
}

func NewMailTrapClient(fromEmail, smtpAddr, smtpSandboxAddr, username, password string, smtpPort int, logger *zap.SugaredLogger) *MailTrapClient {
	return &MailTrapClient{
		smtpAddr:        smtpAddr,
		smtpSandboxAddr: smtpSandboxAddr,
		smtpPort:        smtpPort,
		username:        username,
		password:        password,
		FromEmail:       fromEmail,
		logger:          logger,
	}
}

func (c *MailTrapClient) Send(templateFile, username, email string, data any, isSandbox bool) error {
	message := gomail.NewMessage()

	//template parsing and building

	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)

	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)

	if err != nil {
		return err
	}

	body := new(bytes.Buffer)

	err = tmpl.ExecuteTemplate(body, "body", data)
	if err != nil {
		return err
	}
	// Set email headers
	message.SetHeader("From", c.FromEmail)
	message.SetHeader("To", email)
	message.SetHeader("Subject", subject.String())

	message.SetBody("text/html", body.String())

	smtpAddr := c.smtpAddr

	if isSandbox {
		smtpAddr = c.smtpSandboxAddr
	}

	for i := 0; i <= maxRetries; i++ {
		dialer := gomail.NewDialer(smtpAddr, c.smtpPort, c.username, c.password)
		err := dialer.DialAndSend(message)

		if err == nil {
			if c.logger != nil {
				c.logger.Infow("Email sent", "email", email)
			}
			return nil
		}

		if c.logger != nil {
			c.logger.Errorw("Failed to send email", "email", email, "attempted", i+1, "trials", maxRetries)
			c.logger.Errorf("Error: %v", err.Error())
		}
		//exponential backoff
		time.Sleep(time.Second * time.Duration(i+1))
	}

	return fmt.Errorf("failed to send email after %d attemots", maxRetries)
}

func (c *MailTrapClient) SendEmail(templateFile string,
	to []string,
	cc []string,
	bcc []string,
	attachFiles []string,
) error {
	return nil
}
