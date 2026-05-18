package db_test

import (
	"testing"

	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/testsupport"
)

func TestInsertAndGetBasket(t *testing.T) {
	testsupport.RequireDB(t)

	id := testsupport.InsertBasket(t, "Growth Basket")

	got, err := db.GetBasket(id)
	if err != nil {
		t.Fatalf("GetBasket: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	if got.Name != "Growth Basket" {
		t.Errorf("Name = %q, want %q", got.Name, "Growth Basket")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt was not populated")
	}
}

func TestGetBasketNotFound(t *testing.T) {
	testsupport.RequireDB(t)

	if _, err := db.GetBasket(-1); err == nil {
		t.Error("expected an error for a missing basket, got nil")
	}
}

func TestBasketItems(t *testing.T) {
	testsupport.RequireDB(t)

	basketID := testsupport.InsertBasket(t, "Two Fund Basket")
	f1 := testsupport.InsertFund(t, "Basket Item Fund One")
	f2 := testsupport.InsertFund(t, "Basket Item Fund Two")

	if err := db.InsertBasketItem(basketID, f1, 60); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}
	if err := db.InsertBasketItem(basketID, f2, 40); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	items, err := db.GetBasketItems(basketID)
	if err != nil {
		t.Fatalf("GetBasketItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].FundID != f1 || items[0].Weight != 60 {
		t.Errorf("items[0] = %+v, want fund=%d weight=60", items[0], f1)
	}
	if items[1].FundID != f2 || items[1].Weight != 40 {
		t.Errorf("items[1] = %+v, want fund=%d weight=40", items[1], f2)
	}
}

func TestListBaskets(t *testing.T) {
	testsupport.RequireDB(t)

	id := testsupport.InsertBasket(t, "Listed Basket")

	baskets, err := db.ListBaskets()
	if err != nil {
		t.Fatalf("ListBaskets: %v", err)
	}

	found := false
	for _, b := range baskets {
		if b.ID == id {
			found = true
		}
	}
	if !found {
		t.Errorf("basket %d not present in ListBaskets result", id)
	}
}

func TestDeleteBasket(t *testing.T) {
	testsupport.RequireDB(t)

	id := testsupport.InsertBasket(t, "Doomed Basket")
	fundID := testsupport.InsertFund(t, "Doomed Basket Fund")
	if err := db.InsertBasketItem(id, fundID, 100); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	affected, err := db.DeleteBasket(id)
	if err != nil {
		t.Fatalf("DeleteBasket: %v", err)
	}
	if affected != 1 {
		t.Errorf("first delete affected %d rows, want 1", affected)
	}

	if _, err := db.GetBasket(id); err == nil {
		t.Error("basket still readable after delete")
	}
	if items, _ := db.GetBasketItems(id); len(items) != 0 {
		t.Errorf("%d basket items survived the delete", len(items))
	}

	affected, err = db.DeleteBasket(id)
	if err != nil {
		t.Fatalf("second DeleteBasket: %v", err)
	}
	if affected != 0 {
		t.Errorf("second delete affected %d rows, want 0", affected)
	}
}

func TestSearchFunds(t *testing.T) {
	testsupport.RequireDB(t)

	// A distinctive name token so the search matches only this fund.
	id := testsupport.InsertFund(t, "Zephyr Quantum Index Fund Direct Growth")

	funds, err := db.SearchFunds("Zephyr Quantum", 10, 0)
	if err != nil {
		t.Fatalf("SearchFunds: %v", err)
	}
	if len(funds) != 1 || funds[0].ID != id {
		t.Errorf("SearchFunds returned %+v, want only fund %d", funds, id)
	}
}

func TestFundExists(t *testing.T) {
	testsupport.RequireDB(t)

	id := testsupport.InsertFund(t, "Existing Fund")

	if ok, err := db.FundExists(id); err != nil || !ok {
		t.Errorf("FundExists(%d) = %v, %v; want true, nil", id, ok, err)
	}
	if ok, err := db.FundExists(-1); err != nil || ok {
		t.Errorf("FundExists(-1) = %v, %v; want false, nil", ok, err)
	}
}

func TestNAVHistoryAndCount(t *testing.T) {
	testsupport.RequireDB(t)

	fundID := testsupport.InsertFund(t, "NAV History Fund")
	testsupport.InsertNAV(t, fundID, "2022-01-01", 100)
	testsupport.InsertNAV(t, fundID, "2022-06-01", 110)
	testsupport.InsertNAV(t, fundID, "2023-01-01", 120)

	count, err := db.CountNAVDates(fundID)
	if err != nil {
		t.Fatalf("CountNAVDates: %v", err)
	}
	if count != 3 {
		t.Errorf("CountNAVDates = %d, want 3", count)
	}

	records, err := db.GetNAVHistory(fundID, "2022-01-01", "2022-12-31")
	if err != nil {
		t.Fatalf("GetNAVHistory: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records in range, want 2", len(records))
	}
	if records[0].NAV != 100 || records[1].NAV != 110 {
		t.Errorf("records = %+v, want NAVs 100 then 110", records)
	}
}

func TestMigrationsApplied(t *testing.T) {
	testsupport.RequireDB(t)

	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count < 3 {
		t.Errorf("schema_migrations has %d rows, want at least 3", count)
	}
}

func TestBasketFundIDs(t *testing.T) {
	testsupport.RequireDB(t)

	basketID := testsupport.InsertBasket(t, "Fund IDs Basket")
	f1 := testsupport.InsertFund(t, "Fund IDs Fund One")
	f2 := testsupport.InsertFund(t, "Fund IDs Fund Two")
	if err := db.InsertBasketItem(basketID, f1, 50); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}
	if err := db.InsertBasketItem(basketID, f2, 50); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	ids, err := db.BasketFundIDs()
	if err != nil {
		t.Fatalf("BasketFundIDs: %v", err)
	}

	seen := make(map[int]bool)
	for _, id := range ids {
		seen[id] = true
	}
	if !seen[f1] || !seen[f2] {
		t.Errorf("BasketFundIDs missing %d or %d; got %v", f1, f2, ids)
	}
}

func TestGetFund(t *testing.T) {
	testsupport.RequireDB(t)

	id := testsupport.InsertFund(t, "Lookup Fund")

	fund, err := db.GetFund(id)
	if err != nil {
		t.Fatalf("GetFund: %v", err)
	}
	if fund.ID != id || fund.SchemeName != "Lookup Fund" {
		t.Errorf("GetFund = %+v, want id=%d name=%q", fund, id, "Lookup Fund")
	}
}
