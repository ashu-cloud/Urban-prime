package paymentv1

type AuthorizeHoldRequest struct {
	TripId          string `json:"trip_id,omitempty"`
	RiderId         string `json:"rider_id,omitempty"`
	AmountCents     int64  `json:"amount_cents,omitempty"`
	Currency        string `json:"currency,omitempty"`
	PaymentMethodId string `json:"payment_method_id,omitempty"`
}

func (x *AuthorizeHoldRequest) GetTripId() string          { if x != nil { return x.TripId }; return "" }
func (x *AuthorizeHoldRequest) GetRiderId() string         { if x != nil { return x.RiderId }; return "" }
func (x *AuthorizeHoldRequest) GetAmountCents() int64      { if x != nil { return x.AmountCents }; return 0 }
func (x *AuthorizeHoldRequest) GetCurrency() string         { if x != nil { return x.Currency }; return "" }
func (x *AuthorizeHoldRequest) GetPaymentMethodId() string { if x != nil { return x.PaymentMethodId }; return "" }

type AuthorizeHoldResponse struct {
	Success       bool   `json:"success,omitempty"`
	TransactionId string `json:"transaction_id,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

func (x *AuthorizeHoldResponse) GetSuccess() bool          { if x != nil { return x.Success }; return false }
func (x *AuthorizeHoldResponse) GetTransactionId() string  { if x != nil { return x.TransactionId }; return "" }
func (x *AuthorizeHoldResponse) GetErrorMessage() string   { if x != nil { return x.ErrorMessage }; return "" }

type ReleaseHoldRequest struct {
	TransactionId string `json:"transaction_id,omitempty"`
	TripId        string `json:"trip_id,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func (x *ReleaseHoldRequest) GetTransactionId() string { if x != nil { return x.TransactionId }; return "" }
func (x *ReleaseHoldRequest) GetTripId() string        { if x != nil { return x.TripId }; return "" }
func (x *ReleaseHoldRequest) GetReason() string        { if x != nil { return x.Reason }; return "" }

type ReleaseHoldResponse struct {
	Success bool `json:"success,omitempty"`
}

func (x *ReleaseHoldResponse) GetSuccess() bool { if x != nil { return x.Success }; return false }

type CapturePaymentRequest struct {
	TransactionId    string `json:"transaction_id,omitempty"`
	TripId           string `json:"trip_id,omitempty"`
	FinalAmountCents int64  `json:"final_amount_cents,omitempty"`
}

func (x *CapturePaymentRequest) GetTransactionId() string    { if x != nil { return x.TransactionId }; return "" }
func (x *CapturePaymentRequest) GetTripId() string           { if x != nil { return x.TripId }; return "" }
func (x *CapturePaymentRequest) GetFinalAmountCents() int64  { if x != nil { return x.FinalAmountCents }; return 0 }

type CapturePaymentResponse struct {
	Success    bool   `json:"success,omitempty"`
	ReceiptUrl string `json:"receipt_url,omitempty"`
}

func (x *CapturePaymentResponse) GetSuccess() bool    { if x != nil { return x.Success }; return false }
func (x *CapturePaymentResponse) GetReceiptUrl() string { if x != nil { return x.ReceiptUrl }; return "" }
