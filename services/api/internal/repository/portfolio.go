package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

func (r *ClearflowRepository) EnsureDefaultBrokerageAccount(ctx context.Context, userID string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text FROM brokerage_accounts
		WHERE user_id = $1
		ORDER BY created_at
		LIMIT 1
	`, userID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	now := time.Now().UTC()
	id = auth.NewID()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO brokerage_accounts (id, user_id, provider, account_name, account_type, currency, institution_name, connection_status, created_at, updated_at)
		VALUES ($1, $2, 'manual', 'Demo TFSA', 'TFSA', 'CAD', 'Manual Import', 'manual', $3, $3)
	`, id, userID, now)
	return id, err
}

func (r *ClearflowRepository) CreateBrokerageAccount(ctx context.Context, acct models.BrokerageAccount) (models.BrokerageAccount, error) {
	now := time.Now().UTC()
	acct.ID = fallback(acct.ID, auth.NewID())
	acct.Provider = fallback(acct.Provider, "manual")
	acct.AccountType = fallback(acct.AccountType, "other")
	acct.Currency = fallback(acct.Currency, "USD")
	acct.ConnectionStatus = fallback(acct.ConnectionStatus, "manual")
	acct.CreatedAt = zeroTime(acct.CreatedAt, now)
	acct.UpdatedAt = zeroTime(acct.UpdatedAt, now)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO brokerage_accounts (id, user_id, provider, account_name, account_type, currency, institution_name, connection_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, acct.ID, acct.UserID, acct.Provider, fallback(acct.AccountName, "Manual Portfolio"), acct.AccountType, acct.Currency, acct.InstitutionName, acct.ConnectionStatus, acct.CreatedAt, acct.UpdatedAt)
	return acct, err
}

func (r *ClearflowRepository) ListBrokerageAccounts(ctx context.Context, userID string) ([]models.BrokerageAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, provider, account_name, account_type, currency, COALESCE(institution_name, ''), connection_status, created_at, updated_at
		FROM brokerage_accounts
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.BrokerageAccount{}
	for rows.Next() {
		var acct models.BrokerageAccount
		if err := rows.Scan(&acct.ID, &acct.UserID, &acct.Provider, &acct.AccountName, &acct.AccountType, &acct.Currency, &acct.InstitutionName, &acct.ConnectionStatus, &acct.CreatedAt, &acct.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, acct)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) GetBrokerageAccount(ctx context.Context, userID, accountID string) (models.BrokerageAccount, error) {
	var acct models.BrokerageAccount
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, user_id::text, provider, account_name, account_type, currency, COALESCE(institution_name, ''), connection_status, created_at, updated_at
		FROM brokerage_accounts
		WHERE user_id = $1 AND id = $2
	`, userID, accountID).Scan(&acct.ID, &acct.UserID, &acct.Provider, &acct.AccountName, &acct.AccountType, &acct.Currency, &acct.InstitutionName, &acct.ConnectionStatus, &acct.CreatedAt, &acct.UpdatedAt)
	return acct, err
}

func (r *ClearflowRepository) DeleteBrokerageAccount(ctx context.Context, userID, accountID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM brokerage_accounts WHERE user_id = $1 AND id = $2`, userID, accountID)
	return err
}

func (r *ClearflowRepository) SavePortfolioImport(ctx context.Context, imp models.RawImport, holdings []models.Holding, txs []models.PortfolioTransaction, importErrors []models.ImportError) (models.RawImport, error) {
	now := time.Now().UTC()
	imp.ID = fallback(imp.ID, auth.NewID())
	imp.CreatedAt = zeroTime(imp.CreatedAt, now)
	dbtx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.RawImport{}, err
	}
	defer rollback(dbtx)
	if _, err := dbtx.ExecContext(ctx, `
		INSERT INTO raw_imports (id, user_id, import_type, original_filename, raw_storage_key, row_count, imported_count, failed_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, imp.ID, imp.UserID, imp.ImportType, imp.OriginalFilename, imp.RawStorageKey, imp.RowCount, imp.ImportedCount, imp.FailedCount, imp.CreatedAt); err != nil {
		return models.RawImport{}, err
	}
	for _, importError := range importErrors {
		if importError.ID == "" {
			importError.ID = auth.NewID()
		}
		importError.ImportID = imp.ID
		rawRow, _ := json.Marshal(importError.RawRow)
		if _, err := dbtx.ExecContext(ctx, `
			INSERT INTO import_errors (id, import_id, user_id, row_number, field, code, message, raw_row, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		`, importError.ID, importError.ImportID, importError.UserID, importError.RowNumber, importError.Field, importError.Code, importError.Message, string(rawRow), zeroTime(importError.CreatedAt, now)); err != nil {
			return models.RawImport{}, err
		}
	}
	for _, h := range holdings {
		if h.ID == "" {
			h.ID = auth.NewID()
		}
		if _, err := dbtx.ExecContext(ctx, `
			DELETE FROM holdings WHERE user_id = $1 AND brokerage_account_id = NULLIF($2, '')::uuid AND upper(symbol) = upper($3)
		`, h.UserID, h.BrokerageAccountID, h.Symbol); err != nil {
			return models.RawImport{}, err
		}
		if _, err := dbtx.ExecContext(ctx, `
			INSERT INTO holdings (id, user_id, brokerage_account_id, symbol, security_name, security_type, quantity, average_cost, currency, market_value, last_price, price_as_of, created_at, updated_at)
			VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, h.ID, h.UserID, h.BrokerageAccountID, h.Symbol, h.SecurityName, h.SecurityType, h.Quantity, h.AverageCost, h.Currency, h.MarketValue, h.LastPrice, nullableTime(h.PriceAsOf), zeroTime(h.CreatedAt, now), zeroTime(h.UpdatedAt, now)); err != nil {
			return models.RawImport{}, err
		}
	}
	for _, row := range txs {
		if row.ID == "" {
			row.ID = auth.NewID()
		}
		row.ImportID = imp.ID
		if _, err := dbtx.ExecContext(ctx, `
			INSERT INTO portfolio_transactions (id, user_id, brokerage_account_id, symbol, transaction_type, quantity, price, amount, fees, currency, occurred_at, description, import_id, created_at)
			SELECT $1, $2, NULLIF($3, '')::uuid, NULLIF($4, ''), $5, $6::numeric, $7::numeric, $8::numeric, $9::numeric, $10, $11, $12, $13, $14
			WHERE NOT EXISTS (
				SELECT 1 FROM portfolio_transactions
				WHERE user_id = $2 AND brokerage_account_id IS NOT DISTINCT FROM NULLIF($3, '')::uuid
				  AND COALESCE(symbol, '') = COALESCE(NULLIF($4, ''), '')
				  AND transaction_type = $5 AND occurred_at = $11
				  AND COALESCE(amount, 0) = COALESCE($8::numeric, 0)
				  AND COALESCE(quantity, 0) = COALESCE($6::numeric, 0)
				  AND COALESCE(description, '') = COALESCE($12, '')
			)
		`, row.ID, row.UserID, row.BrokerageAccountID, row.Symbol, row.TransactionType, row.Quantity, row.Price, row.Amount, row.Fees, row.Currency, row.OccurredAt, row.Description, row.ImportID, zeroTime(row.CreatedAt, now)); err != nil {
			return models.RawImport{}, err
		}
	}
	if err := dbtx.Commit(); err != nil {
		return models.RawImport{}, err
	}
	return imp, nil
}

func (r *ClearflowRepository) ListImportErrors(ctx context.Context, userID, importID string) ([]models.ImportError, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, import_id::text, user_id::text, row_number, COALESCE(field, ''), code, message, raw_row::text, created_at
		FROM import_errors
		WHERE user_id = $1 AND import_id = $2
		ORDER BY row_number, created_at
	`, userID, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ImportError{}
	for rows.Next() {
		var importError models.ImportError
		var rawRow string
		if err := rows.Scan(&importError.ID, &importError.ImportID, &importError.UserID, &importError.RowNumber, &importError.Field, &importError.Code, &importError.Message, &rawRow, &importError.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rawRow), &importError.RawRow)
		out = append(out, importError)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) CreateHolding(ctx context.Context, h models.Holding) (models.Holding, error) {
	now := time.Now().UTC()
	h.ID = fallback(h.ID, auth.NewID())
	h.CreatedAt = zeroTime(h.CreatedAt, now)
	h.UpdatedAt = zeroTime(h.UpdatedAt, now)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO holdings (id, user_id, brokerage_account_id, symbol, security_name, security_type, quantity, average_cost, currency, market_value, last_price, price_as_of, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, h.ID, h.UserID, h.BrokerageAccountID, h.Symbol, h.SecurityName, h.SecurityType, h.Quantity, h.AverageCost, h.Currency, h.MarketValue, h.LastPrice, nullableTime(h.PriceAsOf), h.CreatedAt, h.UpdatedAt)
	return h, err
}

func (r *ClearflowRepository) ListHoldings(ctx context.Context, userID string) ([]models.Holding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, COALESCE(brokerage_account_id::text, ''), symbol, COALESCE(security_name, ''), security_type, quantity, COALESCE(average_cost, 0), currency, COALESCE(market_value, 0), COALESCE(last_price, 0), price_as_of, created_at, updated_at
		FROM holdings
		WHERE user_id = $1
		ORDER BY market_value DESC NULLS LAST, symbol
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHoldings(rows)
}

func (r *ClearflowRepository) GetHolding(ctx context.Context, userID, holdingID string) (models.Holding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, COALESCE(brokerage_account_id::text, ''), symbol, COALESCE(security_name, ''), security_type, quantity, COALESCE(average_cost, 0), currency, COALESCE(market_value, 0), COALESCE(last_price, 0), price_as_of, created_at, updated_at
		FROM holdings
		WHERE user_id = $1 AND id = $2
	`, userID, holdingID)
	if err != nil {
		return models.Holding{}, err
	}
	defer rows.Close()
	holdings, err := scanHoldings(rows)
	if err != nil {
		return models.Holding{}, err
	}
	if len(holdings) == 0 {
		return models.Holding{}, sql.ErrNoRows
	}
	return holdings[0], nil
}

func (r *ClearflowRepository) UpdateHolding(ctx context.Context, h models.Holding) (models.Holding, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE holdings
		SET quantity = $3, average_cost = $4, market_value = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
	`, h.UserID, h.ID, h.Quantity, h.AverageCost, h.MarketValue)
	if err != nil {
		return models.Holding{}, err
	}
	return r.GetHolding(ctx, h.UserID, h.ID)
}

func (r *ClearflowRepository) DeleteHolding(ctx context.Context, userID, holdingID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM holdings WHERE user_id = $1 AND id = $2`, userID, holdingID)
	return err
}

func (r *ClearflowRepository) ListPortfolioTransactions(ctx context.Context, userID string) ([]models.PortfolioTransaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, COALESCE(brokerage_account_id::text, ''), COALESCE(symbol, ''), transaction_type, COALESCE(quantity, 0), COALESCE(price, 0), COALESCE(amount, 0), COALESCE(fees, 0), COALESCE(currency, ''), occurred_at, COALESCE(description, ''), COALESCE(import_id::text, ''), created_at
		FROM portfolio_transactions
		WHERE user_id = $1
		ORDER BY occurred_at DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.PortfolioTransaction{}
	for rows.Next() {
		var row models.PortfolioTransaction
		if err := rows.Scan(&row.ID, &row.UserID, &row.BrokerageAccountID, &row.Symbol, &row.TransactionType, &row.Quantity, &row.Price, &row.Amount, &row.Fees, &row.Currency, &row.OccurredAt, &row.Description, &row.ImportID, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ClearflowRepository) ListPortfolioImports(ctx context.Context, userID string) ([]models.RawImport, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, import_type, COALESCE(original_filename, ''), COALESCE(raw_storage_key, ''), row_count, imported_count, failed_count, created_at
		FROM raw_imports
		WHERE user_id = $1 AND import_type IN ('holdings', 'portfolio_transactions')
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.RawImport{}
	for rows.Next() {
		var imp models.RawImport
		if err := rows.Scan(&imp.ID, &imp.UserID, &imp.ImportType, &imp.OriginalFilename, &imp.RawStorageKey, &imp.RowCount, &imp.ImportedCount, &imp.FailedCount, &imp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, imp)
	}
	return out, rows.Err()
}

func scanHoldings(rows *sql.Rows) ([]models.Holding, error) {
	out := []models.Holding{}
	for rows.Next() {
		var h models.Holding
		var priceAsOf sql.NullTime
		if err := rows.Scan(&h.ID, &h.UserID, &h.BrokerageAccountID, &h.Symbol, &h.SecurityName, &h.SecurityType, &h.Quantity, &h.AverageCost, &h.Currency, &h.MarketValue, &h.LastPrice, &priceAsOf, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		if priceAsOf.Valid {
			h.PriceAsOf = priceAsOf.Time
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
