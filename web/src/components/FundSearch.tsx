import { useEffect, useState } from 'react'
import { api, errorMessage } from '../api/client'
import type { Fund } from '../api/types'
import { useDebounce } from '../hooks/useDebounce'
import styles from './FundSearch.module.css'

const PAGE_SIZE = 20

interface Props {
  // When provided, each result shows an "Add" button invoking onAdd.
  onAdd?: (fund: Fund) => void
  // Fund ids already chosen — their "Add" button is disabled.
  addedIds?: number[]
}

// FundSearch is a reusable, debounced, paginated mutual-fund search.
export default function FundSearch({ onAdd, addedIds = [] }: Props) {
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebounce(query, 300)

  const [funds, setFunds] = useState<Fund[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Fetch a fresh first page whenever the debounced query changes.
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    api
      .searchFunds(debouncedQuery, PAGE_SIZE, 0)
      .then((res) => {
        if (cancelled) return
        setFunds(res)
        setOffset(res.length)
        setHasMore(res.length === PAGE_SIZE)
      })
      .catch((e) => {
        if (!cancelled) setError(errorMessage(e))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [debouncedQuery])

  function loadMore() {
    setLoading(true)
    api
      .searchFunds(debouncedQuery, PAGE_SIZE, offset)
      .then((res) => {
        setFunds((prev) => [...prev, ...res])
        setOffset((o) => o + res.length)
        setHasMore(res.length === PAGE_SIZE)
      })
      .catch((e) => setError(errorMessage(e)))
      .finally(() => setLoading(false))
  }

  const added = new Set(addedIds)

  return (
    <div className={styles.search}>
      <input
        className={`input ${styles.queryInput}`}
        type="search"
        placeholder="Search funds by name or scheme code…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {error && <p className="error">{error}</p>}

      <ul className={styles.results}>
        {funds.map((fund) => (
          <li key={fund.id} className={styles.row}>
            <div>
              <div className={styles.name}>{fund.scheme_name}</div>
              <div className={styles.meta}>
                Scheme {fund.scheme_code}
                {fund.fund_house ? ` · ${fund.fund_house}` : ''}
              </div>
            </div>
            {onAdd && (
              <button
                type="button"
                className="btn"
                disabled={added.has(fund.id)}
                onClick={() => onAdd(fund)}
              >
                {added.has(fund.id) ? 'Added' : 'Add'}
              </button>
            )}
          </li>
        ))}
      </ul>

      {!loading && funds.length === 0 && !error && (
        <p className="muted">No funds match that search.</p>
      )}
      {loading && <p className="muted">Loading…</p>}
      {hasMore && !loading && (
        <button type="button" className="btn" onClick={loadMore}>
          Load more
        </button>
      )}
    </div>
  )
}
