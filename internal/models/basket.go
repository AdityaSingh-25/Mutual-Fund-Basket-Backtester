package models

import "time"

type Basket struct {
	ID        int
	Name      string
	CreatedAt time.Time
}
