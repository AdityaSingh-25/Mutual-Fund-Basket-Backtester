import { useEffect, useState } from 'react'
import { api } from '../api/client'
import styles from './Home.module.css'

type Status = 'checking' | 'online' | 'offline'

export default function Home() {
  const [status, setStatus] = useState<Status>('checking')

  useEffect(() => {
    let cancelled = false
    api
      .health()
      .then((res) => {
        if (!cancelled) setStatus(res.status === 'ok' ? 'online' : 'offline')
      })
      .catch(() => {
        if (!cancelled) setStatus('offline')
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <section>
      <h1 className={styles.title}>Mutual Fund Basket Backtester</h1>
      <p className={styles.lede}>
        Build weighted baskets of mutual funds and backtest them against real
        historical NAV data — lump-sum or SIP, with optional rebalancing and a
        benchmark comparison.
      </p>

      <div className={styles.statusCard}>
        <span className={`${styles.dot} ${styles[status]}`} />
        <span>
          Backend API:{' '}
          <strong>{status === 'checking' ? 'checking…' : status}</strong>
        </span>
      </div>

      {status === 'offline' && (
        <p className={styles.hint}>
          The API is unreachable. Start the backend with{' '}
          <code>go run ./cmd/server</code> (or via Docker Compose).
        </p>
      )}
    </section>
  )
}
