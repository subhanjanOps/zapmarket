package domain

// MFARequiredError is returned by LoginPassword when the user has TOTP enabled.
// The HTTP handler converts this to a 200 + MFA_REQUIRED body, not an error response.
type MFARequiredError struct {
	MFASessionToken string
}

func (e *MFARequiredError) Error() string { return "MFA_REQUIRED" }
