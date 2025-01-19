package mailer

import "embed"

var (
	maxRetries          = 3
	UserWelcomeTemplate = "user_invitation.tmpl"
)

//go:embed "templates"
var templateFS embed.FS

type Client interface {
	Send(templateFile, username, email string, data any, isSandbox bool) error
	SendEmail(templateFile string,
		to []string,
		cc []string,
		bcc []string,
		attachFiles []string,
	) error
}
