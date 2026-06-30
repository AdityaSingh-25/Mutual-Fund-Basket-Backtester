package db

import (
	"context"
	"time"

	"MFBasketBacktester/internal/models"
)

// queryTimeout bounds how long any single query may run. It fails a slow or
// stuck query fast instead of letting goroutines pile up under load.
const queryTimeout = 5 * time.Second

// queryCtx returns a context carrying the standard query timeout. Callers must
// defer the returned cancel func.
func queryCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), queryTimeout)
}

func InsertBasket(name string) (int, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var basketID int

	err := DB.QueryRowContext(ctx, `
		INSERT INTO baskets (name)
		VALUES ($1)
		RETURNING id
	`, name).Scan(&basketID)

	return basketID, err
}

func GetBasket(id int) (models.Basket, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var basket models.Basket

	err := DB.QueryRowContext(ctx, `
		SELECT id, name, created_at
		FROM baskets
		WHERE id = $1
	`, id).Scan(
		&basket.ID,
		&basket.Name,
		&basket.CreatedAt,
	)

	return basket, err
}

func ListBaskets() ([]models.Basket, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT id, name, created_at
		FROM baskets
		ORDER BY created_at DESC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baskets []models.Basket

	for rows.Next() {
		var basket models.Basket

		err := rows.Scan(
			&basket.ID,
			&basket.Name,
			&basket.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		baskets = append(baskets, basket)
	}

	return baskets, rows.Err()
}

// DeleteBasket removes a basket and its items in a single transaction. It
// returns the number of basket rows deleted (0 if the basket did not exist).
func DeleteBasket(id int) (int64, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM basket_items WHERE basket_id = $1`, id); err != nil {
		return 0, err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM baskets WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return affected, nil
}

func InsertBasketItem(basketID int, fundID int, weight float64) error {
	ctx, cancel := queryCtx()
	defer cancel()

	_, err := DB.ExecContext(ctx, `
		INSERT INTO basket_items (basket_id, fund_id, weight)
		VALUES ($1, $2, $3)
	`, basketID, fundID, weight)

	return err
}

func GetBasketItems(basketID int) ([]models.BasketItem, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT id, basket_id, fund_id, weight
		FROM basket_items
		WHERE basket_id = $1
		ORDER BY id ASC
	`, basketID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.BasketItem

	for rows.Next() {
		var item models.BasketItem

		err := rows.Scan(
			&item.ID,
			&item.BasketID,
			&item.FundID,
			&item.Weight,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// BasketFundIDs returns the distinct fund ids referenced by any basket — the
// set of funds worth proactively backfilling NAV history for.
func BasketFundIDs() ([]int, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT DISTINCT fund_id
		FROM basket_items
		ORDER BY fund_id ASC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func CountNAVDates(fundID int) (int, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var count int

	err := DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT date)
		FROM nav
		WHERE fund_id = $1
	`, fundID).Scan(&count)

	return count, err
}

func GetFund(id int) (models.Fund, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var fund models.Fund

	err := DB.QueryRowContext(ctx, `
		SELECT id, scheme_code, scheme_name,
		       COALESCE(fund_house, ''), COALESCE(scheme_type, ''), created_at
		FROM funds
		WHERE id = $1
	`, id).Scan(
		&fund.ID,
		&fund.SchemeCode,
		&fund.SchemeName,
		&fund.FundHouse,
		&fund.SchemeType,
		&fund.CreatedAt,
	)

	return fund, err
}

// SearchFunds returns funds whose scheme name contains query, or whose scheme
// code starts with it, ordered by name and paginated by limit/offset.
func SearchFunds(query string, limit, offset int) ([]models.Fund, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT id, scheme_code, scheme_name,
		       COALESCE(fund_house, ''), COALESCE(scheme_type, ''), created_at
		FROM funds
		WHERE scheme_name ILIKE '%' || $1 || '%'
		   OR CAST(scheme_code AS TEXT) LIKE $1 || '%'
		ORDER BY scheme_name ASC
		LIMIT $2 OFFSET $3
	`, query, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funds []models.Fund

	for rows.Next() {
		var fund models.Fund

		err := rows.Scan(
			&fund.ID,
			&fund.SchemeCode,
			&fund.SchemeName,
			&fund.FundHouse,
			&fund.SchemeType,
			&fund.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		funds = append(funds, fund)
	}

	return funds, rows.Err()
}

// FundExists reports whether a fund with the given id is present.
func FundExists(id int) (bool, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	var exists bool

	err := DB.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM funds WHERE id = $1)
	`, id).Scan(&exists)

	return exists, err
}

func GetNAVHistory(fundID int, startDate string, endDate string) ([]models.NAVRecord, error) {
	ctx, cancel := queryCtx()
	defer cancel()

	rows, err := DB.QueryContext(ctx, `
		SELECT date, nav
		FROM nav
		WHERE fund_id = $1
		AND date BETWEEN $2 AND $3
		ORDER BY date ASC
	`, fundID, startDate, endDate)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.NAVRecord

	for rows.Next() {
		var record models.NAVRecord

		err := rows.Scan(
			&record.Date,
			&record.NAV,
		)

		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	return records, nil
}
