package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ClientID string
	Secret   string
	Env      string
	HTTP     *http.Client
}

type LinkTokenResponse struct {
	LinkToken  string `json:"link_token"`
	Expiration string `json:"expiration"`
	RequestID  string `json:"request_id"`
}

type ExchangeResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
	RequestID   string `json:"request_id"`
}

type SandboxPublicTokenResponse struct {
	PublicToken string `json:"public_token"`
	RequestID   string `json:"request_id"`
}

type Institution struct {
	Name string `json:"name"`
}

type ItemGetResponse struct {
	Item struct {
		ItemID        string `json:"item_id"`
		InstitutionID string `json:"institution_id"`
	} `json:"item"`
	Institution Institution `json:"institution"`
}

type Account struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Mask      string `json:"mask"`
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
}

type Transaction struct {
	TransactionID           string   `json:"transaction_id"`
	AccountID               string   `json:"account_id"`
	Amount                  float64  `json:"amount"`
	Date                    string   `json:"date"`
	Name                    string   `json:"name"`
	MerchantName            string   `json:"merchant_name"`
	ISOCurrencyCode         string   `json:"iso_currency_code"`
	Category                []string `json:"category"`
	PersonalFinanceCategory struct {
		Primary  string `json:"primary"`
		Detailed string `json:"detailed"`
	} `json:"personal_finance_category"`
}

type TransactionsSyncResponse struct {
	Added    []Transaction `json:"added"`
	Modified []Transaction `json:"modified"`
	Removed  []struct {
		TransactionID string `json:"transaction_id"`
	} `json:"removed"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
	RequestID  string `json:"request_id"`
}

func (c Client) Ready() bool {
	return c.ClientID != "" && c.Secret != ""
}

func (c Client) CreateLinkToken(ctx context.Context, userID, productsCSV, countriesCSV string) (*LinkTokenResponse, error) {
	payload := map[string]interface{}{
		"client_name":   "Clearflow",
		"user":          map[string]string{"client_user_id": userID},
		"products":      splitCSV(productsCSV),
		"country_codes": splitCSV(countriesCSV),
		"language":      "en",
	}
	var out LinkTokenResponse
	if err := c.post(ctx, "/link/token/create", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c Client) ExchangePublicToken(ctx context.Context, publicToken string) (*ExchangeResponse, error) {
	payload := map[string]interface{}{"public_token": publicToken}
	var out ExchangeResponse
	if err := c.post(ctx, "/item/public_token/exchange", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c Client) CreateSandboxPublicToken(ctx context.Context, institutionID, productsCSV, username, password string) (*SandboxPublicTokenResponse, error) {
	if strings.ToLower(c.Env) != "sandbox" {
		return nil, fmt.Errorf("sandbox public token creation is only available when PLAID_ENV=sandbox")
	}
	if institutionID == "" {
		institutionID = "ins_109508"
	}
	if username == "" {
		username = "user_transactions_dynamic"
	}
	if password == "" {
		password = "pass_good"
	}
	products := splitCSV(productsCSV)
	if len(products) == 0 {
		products = []string{"transactions"}
	}
	payload := map[string]interface{}{
		"institution_id":   institutionID,
		"initial_products": products,
		"options": map[string]string{
			"override_username": username,
			"override_password": password,
		},
	}
	var out SandboxPublicTokenResponse
	if err := c.post(ctx, "/sandbox/public_token/create", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c Client) GetItem(ctx context.Context, accessToken string) (*ItemGetResponse, error) {
	payload := map[string]interface{}{"access_token": accessToken}
	var out ItemGetResponse
	if err := c.post(ctx, "/item/get", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c Client) SyncTransactions(ctx context.Context, accessToken, cursor string) (*TransactionsSyncResponse, error) {
	payload := map[string]interface{}{"access_token": accessToken, "count": 500}
	if cursor != "" {
		payload["cursor"] = cursor
	}
	var out TransactionsSyncResponse
	if err := c.post(ctx, "/transactions/sync", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c Client) post(ctx context.Context, path string, payload map[string]interface{}, out interface{}) error {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	payload["client_id"] = c.ClientID
	payload["secret"] = c.Secret
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		var plaidErr struct {
			ErrorType    string `json:"error_type"`
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		}
		_ = json.NewDecoder(res.Body).Decode(&plaidErr)
		if plaidErr.ErrorMessage == "" {
			plaidErr.ErrorMessage = res.Status
		}
		return fmt.Errorf("plaid %s: %s", plaidErr.ErrorCode, plaidErr.ErrorMessage)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c Client) baseURL() string {
	switch strings.ToLower(c.Env) {
	case "production":
		return "https://production.plaid.com"
	case "development":
		return "https://development.plaid.com"
	default:
		return "https://sandbox.plaid.com"
	}
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
