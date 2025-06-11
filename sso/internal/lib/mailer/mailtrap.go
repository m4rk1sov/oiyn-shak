package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MailtrapClient struct {
	apiToken string
	baseURL  string
	client   *http.Client
}

type MailtrapMessage struct {
	From    MailtrapAddress   `json:"from"`
	To      []MailtrapAddress `json:"to"`
	Subject string            `json:"subject"`
	Text    string            `json:"text,omitempty"`
	HTML    string            `json:"html,omitempty"`
}

type MailtrapAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type MailtrapRequest struct {
	Message MailtrapMessage `json:"message"`
}

func NewMailtrapClient(apiToken string) *MailtrapClient {
	return &MailtrapClient{
		apiToken: apiToken,
		baseURL:  "https://send.api.mailtrap.io/api/send",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *MailtrapClient) SendVerificationEmail(ctx context.Context, toEmail, toName, verificationToken, baseURL string) error {
	verificationURL := fmt.Sprintf("%s/v1/auth/verify-email?token=%s", baseURL, verificationToken)
	
	htmlContent := fmt.Sprintf(`
		<h2>Email Verification</h2>
		<p>Hello %s,</p>
		<p>Please click the link below to verify your email address:</p>
		<p><a href="%s" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
		<p>Or copy and paste this URL in your browser:</p>
		<p>%s</p>
		<p>This link will expire in 24 hours.</p>
		<p>If you didn't create an account, please ignore this email.</p>
	`, toName, verificationURL, verificationURL)
	
	textContent := fmt.Sprintf(`
		Email Verification
		
		Hello %s,
		
		Please open this URL in your browser to verify your email address:
		%s
		
		This link will expire in 24 hours.
		
		If you didn't create an account, please ignore this email.
	`, toName, verificationURL)
	
	message := MailtrapMessage{
		From: MailtrapAddress{
			Email: "noreply@yourapp.com",
			Name:  "Your App",
		},
		To: []MailtrapAddress{
			{
				Email: toEmail,
				Name:  toName,
			},
		},
		Subject: "Please verify your email address",
		HTML:    htmlContent,
		Text:    textContent,
	}
	
	request := MailtrapRequest{Message: message}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiToken)
	
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mailtrap API returned status %d", resp.StatusCode)
	}
	
	return nil
}
