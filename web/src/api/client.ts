import type {
  Basket,
  BacktestRequest,
  BacktestResult,
  CreateBasketRequest,
  Fund,
  SummaryResponse,
} from './types'

const API_BASE =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ??
  'http://localhost:8080'

// ApiError carries the HTTP status alongside the server's error message.
export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })

  const text = await res.text()
  const body: unknown = text ? JSON.parse(text) : null

  if (!res.ok) {
    const message =
      body && typeof body === 'object' && 'error' in body
        ? String((body as { error: unknown }).error)
        : `request failed (${res.status})`
    throw new ApiError(res.status, message)
  }

  return body as T
}

// errorMessage extracts a human-readable string from a thrown value.
export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message
  return 'something went wrong'
}

// api wraps every backtester endpoint with a typed call.
export const api = {
  health: () => request<{ status: string }>('/health'),

  searchFunds: (query: string, limit = 20, offset = 0) =>
    request<Fund[]>(
      `/funds?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`,
    ),
  getFund: (id: number) => request<Fund>(`/funds/${id}`),

  listBaskets: () => request<Basket[]>('/baskets'),
  getBasket: (id: number) => request<Basket>(`/baskets/${id}`),
  createBasket: (body: CreateBasketRequest) =>
    request<Basket>('/baskets', { method: 'POST', body: JSON.stringify(body) }),
  deleteBasket: (id: number) =>
    request<null>(`/baskets/${id}`, { method: 'DELETE' }),

  runBacktest: (body: BacktestRequest) =>
    request<BacktestResult>('/backtest', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  getSummary: (body: BacktestRequest) =>
    request<SummaryResponse>('/summary', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
}
