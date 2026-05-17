package db

import (
	"MFBasketBacktester/internal/models"
)

func InsertBasket(name string) (int, error) {
	var basketID int

	err := DB.QueryRow(`
		INSERT INTO baskets (name)
		VALUES ($1)
		RETURNING id
	`, name).Scan(&basketID)

	return basketID, err
}

func GetBasket(id int) (models.Basket, error) {
	var basket models.Basket

	err := DB.QueryRow(`
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

func InsertBasketItem(basketID int, fundID int, weight float64) error {
	_, err := DB.Exec(`
		INSERT INTO basket_items (basket_id, fund_id, weight)
		VALUES ($1, $2, $3)
	`, basketID, fundID, weight)

	return err
}

func GetBasketItems(basketID int) ([]models.BasketItem, error) {
	rows, err := DB.Query(`
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

func CountNAVDates(fundID int) (int, error) {
	var count int

	err := DB.QueryRow(`
		SELECT COUNT(DISTINCT date)
		FROM nav
		WHERE fund_id = $1
	`, fundID).Scan(&count)

	return count, err
}

func GetFund(id int) (models.Fund, error) {
	var fund models.Fund

	err := DB.QueryRow(`
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

func GetNAVHistory(fundID int, startDate string, endDate string) ([]models.NAVRecord, error) {
	rows, err := DB.Query(`
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
