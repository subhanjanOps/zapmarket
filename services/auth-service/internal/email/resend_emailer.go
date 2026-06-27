package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendEmailer sends transactional emails via the Resend API.
// Free tier: 3 000 emails/month. From address must be "onboarding@resend.dev"
// on the free plan unless a custom domain is verified.
type ResendEmailer struct {
	client *resend.Client
	from   string
}

func NewResendEmailer(apiKey, from string) *ResendEmailer {
	return &ResendEmailer{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

func (e *ResendEmailer) SendPasswordResetEmail(_ context.Context, to, resetLink string) error {
	_, err := e.client.Emails.Send(&resend.SendEmailRequest{
		From:    e.from,
		To:      []string{to},
		Subject: "Reset your ZapMarket password",
		Html: fmt.Sprintf(`<p>You requested a password reset. Click the link below to choose a new password.</p>
<p>This link expires in 30 minutes.</p>
<p><a href="%s">Reset Password</a></p>
<p>If you did not request this, you can safely ignore this email.</p>
<p>— The ZapMarket Team</p>`, resetLink),
	})
	if err != nil {
		return fmt.Errorf("resend: send password reset email: %w", err)
	}
	return nil
}

func (e *ResendEmailer) SendOTPEmail(_ context.Context, to, otp string) error {
	_, err := e.client.Emails.Send(&resend.SendEmailRequest{
		From:    e.from,
		To:      []string{to},
		Subject: "Your ZapMarket verification code",
		Html: fmt.Sprintf(`<p>Your verification code is:</p>
<h2 style="letter-spacing:4px">%s</h2>
<p>This code expires in 10 minutes.</p>
<p>— The ZapMarket Team</p>`, otp),
	})
	if err != nil {
		return fmt.Errorf("resend: send OTP email: %w", err)
	}
	return nil
}
