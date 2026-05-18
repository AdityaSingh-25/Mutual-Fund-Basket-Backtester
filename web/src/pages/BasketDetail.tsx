import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, errorMessage } from '../api/client'
import type { Basket, Fund } from '../api/types'
import pageStyles from './Page.module.css'
import styles from './BasketDetail.module.css'

export default function BasketDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const basketId = Number(id)

  const [basket, setBasket] = useState<Basket | null>(null)
  const [funds, setFunds] = useState<Record<number, Fund>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    api
      .getBasket(basketId)
      .then(async (b) => {
        if (cancelled) return
        setBasket(b)

        // Resolve each item's fund so the table can show real names.
        const fetched = await Promise.all(
          (b.items ?? []).map((it) =>
            api.getFund(it.fund_id).catch(() => null),
          ),
        )
        if (cancelled) return
        const map: Record<number, Fund> = {}
        for (const fund of fetched) if (fund) map[fund.id] = fund
        setFunds(map)
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
  }, [basketId])

  async function remove() {
    try {
      await api.deleteBasket(basketId)
      navigate('/baskets')
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  if (loading) return <p className="muted">Loading…</p>
  if (error) return <p className="error">{error}</p>
  if (!basket) return <p className="error">Basket not found.</p>

  const items = basket.items ?? []

  return (
    <section>
      <div className={pageStyles.headerRow}>
        <h1 className={pageStyles.title}>{basket.name}</h1>
        <button type="button" className="btn btn-danger" onClick={remove}>
          Delete
        </button>
      </div>

      {items.length === 0 ? (
        <p className="muted">This basket has no funds.</p>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Fund</th>
              <th>Scheme</th>
              <th className={styles.num}>Weight</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const fund = funds[item.fund_id]
              return (
                <tr key={item.id}>
                  <td>{fund ? fund.scheme_name : `Fund #${item.fund_id}`}</td>
                  <td>{fund ? fund.scheme_code : '—'}</td>
                  <td className={styles.num}>{item.weight}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </section>
  )
}
