import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, errorMessage } from '../api/client'
import type { Basket } from '../api/types'
import pageStyles from './Page.module.css'
import styles from './Baskets.module.css'

export default function Baskets() {
  const [baskets, setBaskets] = useState<Basket[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .listBaskets()
      .then((res) => {
        if (!cancelled) setBaskets(res)
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
  }, [])

  async function remove(id: number) {
    try {
      await api.deleteBasket(id)
      setBaskets((prev) => prev.filter((b) => b.id !== id))
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  return (
    <section>
      <div className={pageStyles.headerRow}>
        <h1 className={pageStyles.title}>Baskets</h1>
        <Link to="/baskets/new" className="btn btn-primary">
          New basket
        </Link>
      </div>

      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">Loading…</p>}
      {!loading && baskets.length === 0 && (
        <p className="muted">No baskets yet — create your first one.</p>
      )}

      <ul className={styles.list}>
        {baskets.map((basket) => (
          <li key={basket.id} className={styles.row}>
            <Link to={`/baskets/${basket.id}`} className={styles.name}>
              {basket.name}
            </Link>
            <div className={styles.actions}>
              <span className={styles.date}>
                {new Date(basket.created_at).toLocaleDateString()}
              </span>
              <button
                type="button"
                className="btn btn-danger"
                onClick={() => remove(basket.id)}
              >
                Delete
              </button>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
