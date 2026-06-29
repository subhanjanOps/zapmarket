package domain

import "time"

type Review struct {
	ID               string
	ProductID        string
	SKUID            string
	UserID           string
	OrderID          string
	VerifiedPurchase bool
	Rating           int
	Title            string
	Body             string
	ImageURLs        []string
	HelpfulCount     int
	Status           string // PENDING_MODERATION, PUBLISHED, REJECTED
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReturnRequest struct {
	ID          string
	OrderID     string
	OrderItemID string
	UserID      string
	Reason      string // DAMAGED, WRONG_ITEM, NOT_AS_DESCRIBED, CHANGED_MIND
	Description string
	Status      string // REQUESTED, APPROVED, REJECTED, REFUNDED
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
