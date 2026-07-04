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
