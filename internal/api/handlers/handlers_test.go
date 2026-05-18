package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"MFBasketBacktester/internal/api/handlers"
	"MFBasketBacktester/internal/db"
	"MFBasketBacktester/internal/models"
	"MFBasketBacktester/internal/testsupport"
)

// jsonRequest builds an HTTP request with a JSON-encoded body.
func jsonRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateBasketHandler(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	fundID := testsupport.InsertFund(t, "Create Basket Handler Fund")

	body := models.CreateBasketRequest{
		Name:  "API Basket",
		Items: []models.CreateBasketItemInput{{FundID: fundID, Weight: 100}},
	}
	rec := httptest.NewRecorder()
	h.CreateBasket(rec, jsonRequest(t, http.MethodPost, "/baskets", body))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	var got models.Basket
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	t.Cleanup(func() { db.DeleteBasket(got.ID) })

	if got.Name != "API Basket" {
		t.Errorf("Name = %q, want %q", got.Name, "API Basket")
	}
	if len(got.Items) != 1 || got.Items[0].FundID != fundID {
		t.Errorf("Items = %+v, want one item for fund %d", got.Items, fundID)
	}
}

func TestCreateBasketValidation(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	fundID := testsupport.InsertFund(t, "Validation Fund")

	cases := []struct {
		name string
		body models.CreateBasketRequest
	}{
		{
			name: "missing name",
			body: models.CreateBasketRequest{
				Items: []models.CreateBasketItemInput{{FundID: fundID, Weight: 1}},
			},
		},
		{
			name: "no items",
			body: models.CreateBasketRequest{Name: "Empty"},
		},
		{
			name: "non-positive weight",
			body: models.CreateBasketRequest{
				Name:  "Zero Weight",
				Items: []models.CreateBasketItemInput{{FundID: fundID, Weight: 0}},
			},
		},
		{
			name: "duplicate fund",
			body: models.CreateBasketRequest{
				Name: "Dup",
				Items: []models.CreateBasketItemInput{
					{FundID: fundID, Weight: 50},
					{FundID: fundID, Weight: 50},
				},
			},
		},
		{
			name: "nonexistent fund",
			body: models.CreateBasketRequest{
				Name:  "Ghost",
				Items: []models.CreateBasketItemInput{{FundID: 99_999_999, Weight: 100}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.CreateBasket(rec, jsonRequest(t, http.MethodPost, "/baskets", tc.body))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateBasketBadJSON(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	req := httptest.NewRequest(http.MethodPost, "/baskets", strings.NewReader("{not valid"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.CreateBasket(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGetBasketHandler(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	basketID := testsupport.InsertBasket(t, "Gettable Basket")

	t.Run("found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/baskets/"+strconv.Itoa(basketID), nil)
		req.SetPathValue("id", strconv.Itoa(basketID))
		h.GetBasket(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/baskets/99999999", nil)
		req.SetPathValue("id", "99999999")
		h.GetBasket(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/baskets/abc", nil)
		req.SetPathValue("id", "abc")
		h.GetBasket(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestListBasketsHandler(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	basketID := testsupport.InsertBasket(t, "Basket In The List")

	rec := httptest.NewRecorder()
	h.ListBaskets(rec, httptest.NewRequest(http.MethodGet, "/baskets", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var baskets []models.Basket
	if err := json.Unmarshal(rec.Body.Bytes(), &baskets); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	found := false
	for _, b := range baskets {
		if b.ID == basketID {
			found = true
		}
	}
	if !found {
		t.Errorf("basket %d not present in list response", basketID)
	}
}

func TestDeleteBasketHandler(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	basketID := testsupport.InsertBasket(t, "Deletable Basket")

	del := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/baskets/"+strconv.Itoa(basketID), nil)
		req.SetPathValue("id", strconv.Itoa(basketID))
		h.DeleteBasket(rec, req)
		return rec.Code
	}

	if code := del(); code != http.StatusNoContent {
		t.Fatalf("first delete status = %d, want 204", code)
	}
	if code := del(); code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404", code)
	}
}

func TestSearchFundsHandler(t *testing.T) {
	testsupport.RequireDB(t)

	h := handlers.New()
	fundID := testsupport.InsertFund(t, "Zephyr Quantum Handler Fund")

	t.Run("matches by name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.SearchFunds(rec, httptest.NewRequest(http.MethodGet, "/funds?q=Zephyr+Quantum+Handler", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var funds []models.Fund
		if err := json.Unmarshal(rec.Body.Bytes(), &funds); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(funds) != 1 || funds[0].ID != fundID {
			t.Errorf("response = %+v, want only fund %d", funds, fundID)
		}
	})

	t.Run("rejects bad limit", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.SearchFunds(rec, httptest.NewRequest(http.MethodGet, "/funds?limit=abc", nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestSummaryHandlerUsesLocalTemplate(t *testing.T) {
	testsupport.RequireDB(t)
	testsupport.RequireRedis(t)

	h := handlers.New()
	fundID := testsupport.InsertFund(t, "Summary Template Fund")
	testsupport.InsertNAV(t, fundID, "2021-01-01", 100)
	testsupport.InsertNAV(t, fundID, "2022-01-01", 120)

	basketID := testsupport.InsertBasket(t, "Summary Template Basket")
	if err := db.InsertBasketItem(basketID, fundID, 100); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	body := models.BacktestRequest{
		BasketID:  basketID,
		StartDate: "2021-01-01",
		EndDate:   "2022-01-01",
		Amount:    10000,
		Mode:      "lumpsum",
	}

	rec := httptest.NewRecorder()
	h.GetSummary(rec, jsonRequest(t, http.MethodPost, "/summary", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var got struct {
		Basket  string                `json:"basket"`
		Result  models.BacktestResult `json:"result"`
		Summary string                `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	wantSummary := "This basket ended at ₹12000.00 from ₹10000.00 invested, with an XIRR of 20.00%. " +
		"Its worst drawdown was 0.00%, meaning the portfolio fell that much from a prior high during the period."
	if got.Basket != "Summary Template Basket" {
		t.Errorf("Basket = %q, want Summary Template Basket", got.Basket)
	}
	if got.Summary != wantSummary {
		t.Errorf("Summary = %q, want %q", got.Summary, wantSummary)
	}
}

func TestRunBacktestWithBenchmark(t *testing.T) {
	testsupport.RequireDB(t)
	testsupport.RequireRedis(t)

	h := handlers.New()

	basketFund := testsupport.InsertFund(t, "Benchmark Test Basket Fund")
	testsupport.InsertNAV(t, basketFund, "2021-01-01", 100)
	testsupport.InsertNAV(t, basketFund, "2022-01-01", 130)

	benchFund := testsupport.InsertFund(t, "Benchmark Test Index Fund")
	testsupport.InsertNAV(t, benchFund, "2021-01-01", 200)
	testsupport.InsertNAV(t, benchFund, "2022-01-01", 220)

	basketID := testsupport.InsertBasket(t, "Benchmark Test Basket")
	if err := db.InsertBasketItem(basketID, basketFund, 100); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	body := models.BacktestRequest{
		BasketID:        basketID,
		StartDate:       "2021-01-01",
		EndDate:         "2022-01-01",
		Amount:          10000,
		BenchmarkFundID: benchFund,
	}

	rec := httptest.NewRecorder()
	h.RunBacktest(rec, jsonRequest(t, http.MethodPost, "/backtest", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var got models.BacktestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Basket: 100 units at 100, valued at 130 → 13000.
	if got.FinalValue != 13000 {
		t.Errorf("basket FinalValue = %v, want 13000", got.FinalValue)
	}
	if got.Benchmark == nil {
		t.Fatal("Benchmark is nil, want a benchmark result")
	}
	// Benchmark: 50 units at 200, valued at 220 → 11000.
	if got.Benchmark.FinalValue != 11000 {
		t.Errorf("benchmark FinalValue = %v, want 11000", got.Benchmark.FinalValue)
	}
	if got.Benchmark.Benchmark != nil {
		t.Error("nested benchmark should not itself carry a benchmark")
	}
}

func TestRunBacktestBenchmarkNotFound(t *testing.T) {
	testsupport.RequireDB(t)
	testsupport.RequireRedis(t)

	h := handlers.New()

	fund := testsupport.InsertFund(t, "Benchmark NotFound Basket Fund")
	testsupport.InsertNAV(t, fund, "2021-01-01", 100)
	testsupport.InsertNAV(t, fund, "2022-01-01", 110)

	basketID := testsupport.InsertBasket(t, "Benchmark NotFound Basket")
	if err := db.InsertBasketItem(basketID, fund, 100); err != nil {
		t.Fatalf("InsertBasketItem: %v", err)
	}

	body := models.BacktestRequest{
		BasketID:        basketID,
		StartDate:       "2021-01-01",
		EndDate:         "2022-01-01",
		Amount:          10000,
		BenchmarkFundID: 99_999_999,
	}

	rec := httptest.NewRecorder()
	h.RunBacktest(rec, jsonRequest(t, http.MethodPost, "/backtest", body))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a nonexistent benchmark fund", rec.Code)
	}
}

func TestRunBacktestInvalidRebalance(t *testing.T) {
	h := handlers.New()

	body := models.BacktestRequest{
		BasketID:  1,
		StartDate: "2021-01-01",
		EndDate:   "2022-01-01",
		Amount:    10000,
		Rebalance: "weekly",
	}

	rec := httptest.NewRecorder()
	h.RunBacktest(rec, jsonRequest(t, http.MethodPost, "/backtest", body))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an invalid rebalance period", rec.Code)
	}
}
