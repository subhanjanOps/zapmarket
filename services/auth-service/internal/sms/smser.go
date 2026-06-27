// Package sms defines the SMSer interface and its concrete implementations.
package sms

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSer sends transactional SMS messages.
type SMSer interface {
	SendOTP(ctx context.Context, to, otp string) error
}

// LogSMSer logs SMS messages instead of sending them. Used when Twilio is not configured.
type LogSMSer struct{}

func NewLogSMSer() *LogSMSer { return &LogSMSer{} }

func (s *LogSMSer) SendOTP(_ context.Context, to, otp string) error {
	slog.Info("OTP SMS (dev mode — not sent)", "to", to, "otp", otp)
	return nil
}

// TwilioSMSer sends SMS messages via Twilio.
type TwilioSMSer struct {
	client *twilio.RestClient
	from   string
}

func NewTwilioSMSer(accountSID, authToken, from string) *TwilioSMSer {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &TwilioSMSer{client: client, from: from}
}

func (s *TwilioSMSer) SendOTP(_ context.Context, to, otp string) error {
	body := fmt.Sprintf("Your ZapMarket verification code is: %s. Expires in 10 minutes.", otp)
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(s.from)
	params.SetBody(body)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio: send OTP to %s: %w", to, err)
	}
	return nil
}
