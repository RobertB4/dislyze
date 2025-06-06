package sendgridlib

const (
	SendGridFromName  = "dislyze"
	SendGridFromEmail = "support@dislyze.com"
)

type SendGridEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SendGridPersonalization struct {
	To      []SendGridEmailAddress `json:"to"`
	Subject string                 `json:"subject"`
}

type SendGridMailRequestBody struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmailAddress      `json:"from"`
	Content          []SendGridContent         `json:"content"`
}
