package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type AdvisorProfile struct {
	ID                         string    `json:"id"`
	UserID                     string    `json:"user_id"`
	Country                    string    `json:"country"`
	Age                        int       `json:"age"`
	MonthlyIncomeEstimate      float64   `json:"monthly_income_estimate"`
	RiskTolerance              string    `json:"risk_tolerance"`
	EmergencyFundMonthsTarget  int       `json:"emergency_fund_months_target"`
	CurrentEmergencyFund       float64   `json:"current_emergency_fund"`
	HasHighInterestDebt        bool      `json:"has_high_interest_debt"`
	HighInterestDebtAmount     float64   `json:"high_interest_debt_amount"`
	HighInterestDebtAPR        float64   `json:"high_interest_debt_apr"`
	HasEmployerMatch           bool      `json:"has_employer_match"`
	EmployerMatchDescription   string    `json:"employer_match_description"`
	RetirementAccountAccess    string    `json:"retirement_account_access"`
	PrimaryGoal                string    `json:"primary_goal"`
	InvestmentTimeHorizonYears int       `json:"investment_time_horizon_years"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type RawImport struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	ImportType       string    `json:"import_type"`
	OriginalFilename string    `json:"original_filename"`
	RawStorageKey    string    `json:"raw_storage_key"`
	RowCount         int       `json:"row_count"`
	ImportedCount    int       `json:"imported_count"`
	FailedCount      int       `json:"failed_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type Transaction struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	AccountID          string                 `json:"account_id"`
	Amount             float64                `json:"amount"`
	Direction          string                 `json:"direction"`
	Currency           string                 `json:"currency"`
	Merchant           string                 `json:"merchant"`
	NormalizedMerchant string                 `json:"normalized_merchant"`
	Category           string                 `json:"category"`
	Description        string                 `json:"description"`
	OccurredAt         time.Time              `json:"occurred_at"`
	RawEventKey        string                 `json:"raw_event_key"`
	ImportID           string                 `json:"import_id"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

type BrokerageAccount struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Provider         string    `json:"provider"`
	AccountName      string    `json:"account_name"`
	AccountType      string    `json:"account_type"`
	Currency         string    `json:"currency"`
	InstitutionName  string    `json:"institution_name"`
	ConnectionStatus string    `json:"connection_status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Holding struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	BrokerageAccountID string    `json:"brokerage_account_id"`
	Symbol             string    `json:"symbol"`
	SecurityName       string    `json:"security_name"`
	SecurityType       string    `json:"security_type"`
	Quantity           float64   `json:"quantity"`
	AverageCost        float64   `json:"average_cost"`
	Currency           string    `json:"currency"`
	MarketValue        float64   `json:"market_value"`
	LastPrice          float64   `json:"last_price"`
	PriceAsOf          time.Time `json:"price_as_of"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PortfolioTransaction struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	BrokerageAccountID string    `json:"brokerage_account_id"`
	Symbol             string    `json:"symbol"`
	TransactionType    string    `json:"transaction_type"`
	Quantity           float64   `json:"quantity"`
	Price              float64   `json:"price"`
	Amount             float64   `json:"amount"`
	Fees               float64   `json:"fees"`
	Currency           string    `json:"currency"`
	OccurredAt         time.Time `json:"occurred_at"`
	Description        string    `json:"description"`
	ImportID           string    `json:"import_id"`
	CreatedAt          time.Time `json:"created_at"`
}

type Recommendation struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Severity  string                 `json:"severity"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type PlaidConnection struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	ItemID                string    `json:"item_id"`
	InstitutionName       string    `json:"institution_name"`
	AccessTokenCiphertext string    `json:"-"`
	Cursor                string    `json:"cursor,omitempty"`
	Products              []string  `json:"products"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	LastSyncedAt          time.Time `json:"last_synced_at,omitempty"`
}

type Organization struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Customer struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
}

type Invoice struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	CustomerID     string    `json:"customer_id"`
	Number         string    `json:"number"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	DueAt          time.Time `json:"due_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type Payment struct {
	ID                 string    `json:"id"`
	OrganizationID     string    `json:"organization_id"`
	Processor          string    `json:"processor"`
	ProcessorPaymentID string    `json:"processor_payment_id"`
	CustomerEmail      string    `json:"customer_email"`
	Amount             float64   `json:"amount"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	OccurredAt         time.Time `json:"occurred_at"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"created_at"`
}

type Refund struct {
	ID                string    `json:"id"`
	OrganizationID    string    `json:"organization_id"`
	ProcessorRefundID string    `json:"processor_refund_id"`
	PaymentID         string    `json:"payment_id"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	OccurredAt        time.Time `json:"occurred_at"`
}

type Fee struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ProcessorFeeID string    `json:"processor_fee_id"`
	PaymentID      string    `json:"payment_id"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	OccurredAt     time.Time `json:"occurred_at"`
	Description    string    `json:"description"`
}

type Payout struct {
	ID                string    `json:"id"`
	OrganizationID    string    `json:"organization_id"`
	Processor         string    `json:"processor"`
	ProcessorPayoutID string    `json:"processor_payout_id"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	ExpectedArrivalAt time.Time `json:"expected_arrival_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type BankTransaction struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Source         string    `json:"source"`
	ExternalID     string    `json:"external_id"`
	Amount         float64   `json:"amount"`
	Direction      string    `json:"direction"`
	Currency       string    `json:"currency"`
	Description    string    `json:"description"`
	PostedAt       time.Time `json:"posted_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReconciliationRun struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Status         string    `json:"status"`
	MatchedCount   int       `json:"matched_count"`
	ExceptionCount int       `json:"exception_count"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

type ReconciliationMatch struct {
	ID                string    `json:"id"`
	OrganizationID    string    `json:"organization_id"`
	RunID             string    `json:"run_id"`
	PayoutID          string    `json:"payout_id"`
	BankTransactionID string    `json:"bank_transaction_id"`
	Amount            float64   `json:"amount"`
	Confidence        float64   `json:"confidence"`
	Explanation       string    `json:"explanation"`
	CreatedAt         time.Time `json:"created_at"`
}

type ReconciliationException struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	RunID          string    `json:"run_id"`
	Type           string    `json:"type"`
	Severity       string    `json:"severity"`
	Title          string    `json:"title"`
	Explanation    string    `json:"explanation"`
	Status         string    `json:"status"`
	ReferenceID    string    `json:"reference_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type AuditLog struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Action         string    `json:"action"`
	TargetType     string    `json:"target_type"`
	TargetID       string    `json:"target_id"`
	CreatedAt      time.Time `json:"created_at"`
}
