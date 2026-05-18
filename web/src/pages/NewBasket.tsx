import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import FundSearch from '../components/FundSearch'
import { api, errorMessage } from '../api/client'
import type { Fund } from '../api/types'
import pageStyles from './Page.module.css'
import styles from './NewBasket.module.css'

interface SelectedFund {
  fund: Fund
  weight: number
}

export default function NewBasket() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<SelectedFund[]>([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function addFund(fund: Fund) {
    setSelected((prev) =>
      prev.some((s) => s.fund.id === fund.id)
        ? prev
        : [...prev, { fund, weight: 1 }],
    )
  }

  function removeFund(id: number) {
    setSelected((prev) => prev.filter((s) => s.fund.id !== id))
  }

  function setWeight(id: number, weight: number) {
    setSelected((prev) =>
      prev.map((s) => (s.fund.id === id ? { ...s, weight } : s)),
    )
  }

  const valid =
    name.trim() !== '' &&
    selected.length > 0 &&
    selected.every((s) => s.weight > 0)

  async function save() {
    setSaving(true)
    setError(null)
    try {
      const basket = await api.createBasket({
        name: name.trim(),
        items: selected.map((s) => ({ fund_id: s.fund.id, weight: s.weight })),
      })
      navigate(`/baskets/${basket.id}`)
    } catch (e) {
      setError(errorMessage(e))
      setSaving(false)
    }
  }

  return (
    <section>
      <h1 className={pageStyles.title}>New basket</h1>

      <label className={styles.field}>
        <span className={styles.label}>Basket name</span>
        <input
          className="input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Nifty 50 Core"
        />
      </label>

      <h2 className={styles.heading}>Selected funds</h2>
      {selected.length === 0 ? (
        <p className="muted">Add funds from the search below.</p>
      ) : (
        <ul className={styles.picks}>
          {selected.map((s) => (
            <li key={s.fund.id} className={styles.pick}>
              <span className={styles.pickName}>{s.fund.scheme_name}</span>
              <input
                className={`input ${styles.weight}`}
                type="number"
                min="0"
                step="any"
                value={s.weight}
                onChange={(e) => setWeight(s.fund.id, Number(e.target.value))}
              />
              <button
                type="button"
                className="btn btn-danger"
                onClick={() => removeFund(s.fund.id)}
              >
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="muted">Weights are relative — they need not sum to 100.</p>

      {error && <p className="error">{error}</p>}

      <div className={styles.actions}>
        <button
          type="button"
          className="btn btn-primary"
          disabled={!valid || saving}
          onClick={save}
        >
          {saving ? 'Saving…' : 'Create basket'}
        </button>
      </div>

      <h2 className={styles.heading}>Add funds</h2>
      <FundSearch onAdd={addFund} addedIds={selected.map((s) => s.fund.id)} />
    </section>
  )
}
