package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type TwilioNotifier struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *http.Client
}

func NewTwilioNotifier(accountSID, authToken, fromNumber string) *TwilioNotifier {
	return &TwilioNotifier{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client:     &http.Client{},
	}
}

func (t *TwilioNotifier) SendSMS(ctx context.Context, phone, message string) error {
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	data := url.Values{
		"To":   {phone},
		"From": {t.fromNumber},
		"Body": {message},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio returned %d", resp.StatusCode)
	}
	return nil
}
