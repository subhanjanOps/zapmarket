package kafka

// Topic name constants — single source of truth across all services.
const (
	TopicOrders    = "orders"
	TopicPayments  = "payments"
	TopicInventory = "inventory"

	// Checkout saga topics
	TopicCheckoutRequested          = "checkout.requested"
	TopicInventoryReserved          = "inventory.reserved"
	TopicInventoryReservationFailed = "inventory.reservation_failed"
	TopicPaymentCaptured            = "payment.captured"
	TopicPaymentFailed              = "payment.failed"
	TopicOrderConfirmed             = "order.confirmed"
	TopicOrderCancelled             = "order.cancelled"
	TopicShipmentDelivered          = "shipment.delivered"
	TopicShipmentUndelivered        = "shipment.undelivered"
	TopicReservationExpired         = "reservation.expired"
)
