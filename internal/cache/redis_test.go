package cache_test

import (
	"reflect"
	"testing"

	"MFBasketBacktester/internal/cache"
	"MFBasketBacktester/internal/models"
	"MFBasketBacktester/internal/testsupport"
)

func TestBacktestResultRoundTrip(t *testing.T) {
	testsupport.RequireRedis(t)

	const key = "test:backtest:roundtrip"
	want := models.BacktestResult{
		CAGR:       12.5,
		XIRR:       12.1,
		Drawdown:   8.4,
		FinalValue: 56789.01,
	}

	if err := cache.SetBacktestResult(key, want); err != nil {
		t.Fatalf("SetBacktestResult: %v", err)
	}
	t.Cleanup(func() { cache.RDB.Del(cache.Ctx, key) })

	got, err := cache.GetBacktestResult(key)
	if err != nil {
		t.Fatalf("GetBacktestResult: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round-trip = %+v, want %+v", *got, want)
	}
}

func TestGetBacktestResultMiss(t *testing.T) {
	testsupport.RequireRedis(t)

	if _, err := cache.GetBacktestResult("test:backtest:definitely-absent"); err == nil {
		t.Error("expected an error for a missing cache key")
	}
}

func TestSummaryRoundTrip(t *testing.T) {
	testsupport.RequireRedis(t)

	const key = "test:summary:roundtrip"
	const want = "Your basket grew steadily over the period."

	if err := cache.SetSummary(key, want); err != nil {
		t.Fatalf("SetSummary: %v", err)
	}
	t.Cleanup(func() { cache.RDB.Del(cache.Ctx, key) })

	got, err := cache.GetSummary(key)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if got != want {
		t.Errorf("GetSummary = %q, want %q", got, want)
	}
}
