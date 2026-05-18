import { NavLink, Outlet } from 'react-router-dom'
import styles from './Layout.module.css'

const links = [
  { to: '/', label: 'Home', end: true },
  { to: '/funds', label: 'Funds', end: false },
  { to: '/baskets', label: 'Baskets', end: false },
  { to: '/backtest', label: 'Backtest', end: false },
]

export default function Layout() {
  return (
    <div className={styles.app}>
      <header className={styles.header}>
        <span className={styles.brand}>MF Basket Backtester</span>
        <nav className={styles.nav}>
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) =>
                isActive ? `${styles.link} ${styles.active}` : styles.link
              }
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}
