import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api, errorMessage } from '../api/client'
import type {
  BacktestResult,
  Basket,
  Mode,
  Rebalance,
} from '../api/types'
import pageStyles from './Page.module.css'
import styles from './Backtest.module.css'

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

function yearsAgoISO(years: number): string {
  const d = new Date()
  d.setFullYear(d.getFullYear() - years)
  return d.toISOString().slice(0, 10)
}

function formatINR(value: number): string {
  return `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`
}

const modeLabel: Record<string, string> = {
  lumpsum: 'Lump sum',
  sip: 'SIP (monthly)',
}

const rebalanceLabel: Record<string, string> = {
  none: 'None',
  monthly: 'Monthly',
  quarterly: 'Quarterly',
  yearly: 'Yearly',
}

export default function Backtest() {
  const [baskets, setBaskets] = useState<Basket[]>([])
  const [basketsError, setBasketsError] = useState<string | null>(null)
  const [basketId, setBasketId] = useState<number | null>(null)

  const [startDate, setStartDate] = useState(yearsAgoISO(3))
  const [endDate, setEndDate] = useState(todayISO())
  const [amount, setAmount] = useState(100000)
  const [mode, setMode] = useState<Mode>('lumpsum')
  const [rebalance, setRebalance] = useState<Rebalance>('none')

  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<BacktestResult | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .listBaskets()
      .then((res) => {
        if (cancelled) return
        setBaskets(res)
        if (res.length > 0) setBasketId(res[0].id)
      })
      .catch((e) => {
        if (!cancelled) setBasketsError(errorMessage(e))
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function run() {
    if (basketId == null) return
    setRunning(true)
    setError(null)
    setResult(null)
    try {
      const res = await api.runBacktest({
        basket_id: basketId,
        start_date: startDate,
        end_date: endDate,
        amount,
        mode,
        rebalance,
      })
      setResult(res)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setRunning(false)
    }
  }

  const formValid =
    basketId != null && startDate < endDate && amount > 0

  return (
    <section>
      <h1 className={pageStyles.title}>Backtest</h1>

      {basketsError && <p className="error">{basketsError}</p>}
      {!basketsError && baskets.length === 0 && (
        <p className="muted">
          You don't have any baskets yet —{' '}
          <Link to="/baskets/new">create one</Link> first.
        </p>
      )}

      {baskets.length > 0 && (
        <>
          <div className={styles.form}>
            <label className={styles.field}>
              <span className={styles.label}>Basket</span>
              <select
                className="input"
                value={basketId ?? ''}
                onChange={(e) => setBasketId(Number(e.target.value))}
              >
                {baskets.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </label>

            <div className={styles.row}>
              <label className={styles.field}>
                <span className={styles.label}>Start date</span>
                <input
                  className="input"
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.label}>End date</span>
                <input
                  className="input"
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Amount (₹)</span>
                <input
                  className="input"
                  type="number"
                  min={1}
                  step="any"
                  value={amount}
                  onChange={(e) => setAmount(Number(e.target.value))}
                />
              </label>
            </div>

            <div className={styles.row}>
              <label className={styles.field}>
                <span className={styles.label}>Mode</span>
                <select
                  className="input"
                  value={mode}
                  onChange={(e) => setMode(e.target.value as Mode)}
                >
                  <option value="lumpsum">Lump sum</option>
                  <option value="sip">SIP (monthly)</option>
                </select>
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Rebalance</span>
                <select
                  className="input"
                  value={rebalance}
                  onChange={(e) => setRebalance(e.target.value as Rebalance)}
                >
                  <option value="none">None</option>
                  <option value="monthly">Monthly</option>
                  <option value="quarterly">Quarterly</option>
                  <option value="yearly">Yearly</option>
                </select>
              </label>
            </div>

            <p className="muted">
              For SIP, the amount is invested every month. The first backtest of a
              basket may take a few seconds while its NAV history is backfilled.
            </p>

            <div>
              <button
                type="button"
                className="btn btn-primary"
                disabled={!formValid || running}
                onClick={run}
              >
                {running ? 'Running…' : 'Run backtest'}
              </button>
            </div>
          </div>

          {error && <p className="error">{error}</p>}
          {result && <BacktestResultView result={result} />}
        </>
      )}
    </section>
  )
}

function BacktestResultView({ result }: { result: BacktestResult }) {
  return (
    <div className={styles.result}>
      <div className={styles.metrics}>
        <Metric label="CAGR" value={`${result.cagr.toFixed(2)}%`} />
        <Metric label="XIRR" value={`${result.xirr.toFixed(2)}%`} />
        <Metric label="Max drawdown" value={`${result.drawdown.toFixed(2)}%`} />
        <Metric label="Total invested" value={formatINR(result.total_invested)} />
        <Metric label="Final value" value={formatINR(result.final_value)} />
        <Metric
          label="Strategy"
          value={`${modeLabel[result.mode] ?? result.mode} · ${rebalanceLabel[result.rebalance] ?? result.rebalance} rebalance`}
        />
      </div>

      {result.series && result.series.length > 0 && (
        <div className={styles.chart}>
          <ResponsiveContainer width="100%" height={320}>
            <LineChart
              data={result.series}
              margin={{ top: 8, right: 16, left: 0, bottom: 0 }}
            >
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} minTickGap={48} />
              <YAxis
                tick={{ fontSize: 11 }}
                width={72}
                tickFormatter={(v: number) =>
                  v.toLocaleString('en-IN', { maximumFractionDigits: 0 })
                }
              />
              <Tooltip
                formatter={(value) =>
                  formatINR(typeof value === 'number' ? value : Number(value))
                }
                labelFormatter={(label) => String(label)}
              />
              <Line
                type="monotone"
                dataKey="value"
                stroke="var(--accent)"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className={styles.metric}>
      <div className={styles.metricLabel}>{label}</div>
      <div className={styles.metricValue}>{value}</div>
    </div>
  )
}
