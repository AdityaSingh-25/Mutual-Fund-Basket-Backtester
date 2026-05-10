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
