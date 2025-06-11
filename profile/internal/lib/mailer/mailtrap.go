package mailer

//
//import (
//	"bytes"
//	"context"
//	"fmt"
//	"net/smtp"
//	"text/template"
//)
//
//// Mailer holds SMTP configuration.
//type Mailer struct {
//	Host     string
//	Port     string
//	Username string
//	Password string
//	From     string
//	FromName string
//}
//
//// New creates a Mailer.
//func New(host, port, user, pass, from, fromName string) *Mailer {
//	return &Mailer{
//		Host:     host,
//		Port:     port,
//		Username: user,
//		Password: pass,
//		From:     from,
//		FromName: fromName,
//	}
//}
//
//// sendEmail sends an email with the given subject and body (HTML or plain-text).
//func (m *Mailer) sendEmail(to, subject, body string) error {
//	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
//	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
//
//	// Build MIME headers
//	msg := bytes.Buffer{}
//	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", m.FromName, m.From))
//	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
//	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
//	msg.WriteString("MIME-Version: 1.0\r\n")
//	msg.WriteString(`Content-Type: text/html; charset="UTF-8"` + "\r\n")
//	msg.WriteString("\r\n")
//	msg.WriteString(body)
//
//	return smtp.SendMail(addr, auth, m.From, []string{to}, msg.Bytes())
//}
//
//// SendVerificationEmail builds and sends a verification email.
//func (m *Mailer) SendVerificationEmail(ctx context.Context, toEmail, toName, verificationToken, baseURL string) error {
//	verificationURL := fmt.Sprintf("%s/v1/auth/verification/confirm/%s", baseURL, verificationToken)
//
//	subject := "Verify your email address"
//	tpl := `
//		<h2>Hello {{.Name}},</h2>
//		<p>Please verify your email by clicking the link below:</p>
//		<p><a href="{{.URL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
//		<p>Or copy and paste this URL in your browser:</p>
//		<p>{{.URL}}</p>
//		<p>This link will expire in 24 hours.</p>
//		<p>If you did not sign up, ignore this email.</p>
//	`
//	data := struct {
//		Name string
//		URL  string
//	}{toName, verificationURL}
//
//	tmpl, err := template.New("verify").Parse(tpl)
//	if err != nil {
//		return fmt.Errorf("failed to parse template: %w", err)
//	}
//
//	var body bytes.Buffer
//	if err := tmpl.Execute(&body, data); err != nil {
//		return fmt.Errorf("failed to execute template: %w", err)
//	}
//
//	return m.sendEmail(toEmail, subject, body.String())
//}
//
//// SendResetPasswordEmail builds and sends a password-reset email.
//func (m *Mailer) SendResetPasswordEmail(ctx context.Context, toEmail, toName, resetToken, baseURL string) error {
//	resetURL := fmt.Sprintf("%s/v1/auth/reset-password?token=%s", baseURL, resetToken)
//
//	subject := "Reset your password"
//	tpl := `
//		<h2>Hello {{.Name}},</h2>
//		<p>You can reset your password by taking the link below with the reset token and updating your password via POST request:</p>
//		<p>Copy and paste this URL</p>
//		<p>{{.URL}}</p>
//		<p>This link will expire in 1 hour.</p>
//		<p>If you did not request a reset, ignore this email.</p>
//	`
//	data := struct {
//		Name string
//		URL  string
//	}{toName, resetURL}
//
//	tmpl, err := template.New("reset").Parse(tpl)
//	if err != nil {
//		return fmt.Errorf("failed to parse template: %w", err)
//	}
//
//	var body bytes.Buffer
//	if err := tmpl.Execute(&body, data); err != nil {
//		return fmt.Errorf("failed to execute template: %w", err)
//	}
//
//	return m.sendEmail(toEmail, subject, body.String())
//}
//
//// Обратная совместимость - создайте псевдоним для старого интерфейса
//type MailtrapClient = Mailer
//
//// NewMailtrapClient создает новый SMTP клиент (для обратной совместимости)
//func NewMailtrapClient(host, port, user, pass, from, fromName string) *MailtrapClient {
//	return New(host, port, user, pass, from, fromName)
//}
//
////package mailer
////
////import (
////	"bytes"
////	"context"
////	"encoding/json"
////	"fmt"
////	"io"
////	"net/http"
////	"time"
////)
////
////type MailtrapClient struct {
////	apiToken string
////	baseURL  string
////	client   *http.Client
////}
////
////type MailtrapAddress struct {
////	Email string `json:"email"`
////	Name  string `json:"name,omitempty"`
////}
////
////type MailtrapRequest struct {
////	From    MailtrapAddress   `json:"from"`
////	To      []MailtrapAddress `json:"to"`
////	Subject string            `json:"subject"`
////	Text    string            `json:"text,omitempty"`
////	HTML    string            `json:"html,omitempty"`
////}
////
////func NewMailtrapClient(apiToken string) *MailtrapClient {
////	return &MailtrapClient{
////		apiToken: apiToken,
////		baseURL:  "https://send.api.mailtrap.io/api/send",
////		client: &http.Client{
////			Timeout: 30 * time.Second,
////		},
////	}
////}
////
////func (m *MailtrapClient) SendVerificationEmail(ctx context.Context, toEmail, toName, verificationToken, baseURL string) error {
////	verificationURL := fmt.Sprintf("%s/v1/auth/verification/confirm?token=%s", baseURL, verificationToken)
////
////	htmlContent := fmt.Sprintf(`
////		<h2>Email Verification</h2>
////		<p>Hello %s,</p>
////		<p>Please click the link below to verify your email address:</p>
////		<p><a href="%s" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
////		<p>Or copy and paste this URL in your browser:</p>
////		<p>%s</p>
////		<p>This link will expire in 24 hours.</p>
////		<p>If you didn't create an account, please ignore this email.</p>
////	`, toName, verificationURL, verificationURL)
////
////	textContent := fmt.Sprintf(`
////		Email Verification
////
////		Hello %s,
////
////		Please open this URL in your browser to verify your email address:
////		%s
////
////		This link will expire in 24 hours.
////
////		If you didn't create an account, please ignore this email.
////	`, toName, verificationURL)
////
////	request := MailtrapRequest{
////		From: MailtrapAddress{
////			Email: "hello@oiyn-shak.com",
////			Name:  "SSO Service",
////		},
////		To: []MailtrapAddress{
////			{
////				Email: toEmail,
////				Name:  toName,
////			},
////		},
////		Subject: "Please verify your email address",
////		HTML:    htmlContent,
////		Text:    textContent,
////	}
////
////	jsonData, err := json.Marshal(request)
////	if err != nil {
////		return fmt.Errorf("failed to marshal request: %w", err)
////	}
////
////	fmt.Printf("Sending request to Mailtrap: %s\n", string(jsonData))
////
////	req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL, bytes.NewBuffer(jsonData))
////	if err != nil {
////		return fmt.Errorf("failed to create request: %w", err)
////	}
////
////	req.Header.Set("Content-Type", "application/json")
////	req.Header.Set("Authorization", "Bearer "+m.apiToken)
////
////	resp, err := m.client.Do(req)
////	if err != nil {
////		return fmt.Errorf("failed to send request: %w", err)
////	}
////	defer func(Body io.ReadCloser) {
////		err := Body.Close()
////		if err != nil {
////			return
////		}
////	}(resp.Body)
////
////	body, _ := io.ReadAll(resp.Body)
////
////	if resp.StatusCode != http.StatusOK {
////		return fmt.Errorf("mailtrap API returned status %d: %s", resp.StatusCode, string(body))
////	}
////
////	fmt.Printf("Email sent successfully: %s\n", string(body))
////	return nil
////}
