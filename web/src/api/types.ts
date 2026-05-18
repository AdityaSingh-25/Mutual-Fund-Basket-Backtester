// TypeScript mirrors of the Go API models (see internal/models).

export interface Fund {
  id: number
  scheme_code: number
  scheme_name: string
  fund_house?: string
  scheme_type?: string
  created_at: string
}

export interface BasketItem {
  id: number
  basket_id: number
  fund_id: number
  weight: number
}

export interface Basket {
  id: number
  name: string
  created_at: string
  items?: BasketItem[]
}

export interface CreateBasketItemInput {
  fund_id: number
  weight: number
}

export interface CreateBasketRequest {
  name: string
  items: CreateBasketItemInput[]
}

export type Mode = 'lumpsum' | 'sip'
export type Rebalance = 'none' | 'monthly' | 'quarterly' | 'yearly'

export interface BacktestRequest {
  basket_id: number
  start_date: string // YYYY-MM-DD
  end_date: string // YYYY-MM-DD
  amount: number
  mode?: Mode
  rebalance?: Rebalance
  benchmark_fund_id?: number
}

export interface SeriesPoint {
  date: string
  value: number
}

export interface BacktestResult {
  mode: string
  rebalance: string
  cagr: number
  xirr: number
  drawdown: number
  total_invested: number
  final_value: number
  series?: SeriesPoint[]
  benchmark?: BacktestResult
}

export interface SummaryResponse {
  basket: string
  result: BacktestResult
  summary: string
}
