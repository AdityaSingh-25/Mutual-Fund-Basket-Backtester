import FundSearch from '../components/FundSearch'
import styles from './Page.module.css'

export default function Funds() {
  return (
    <section>
      <h1 className={styles.title}>Funds</h1>
      <p className={styles.lede}>
        Browse the mutual funds ingested from AMFI. Use the IDs and names here
        when building a basket.
      </p>
      <FundSearch />
    </section>
  )
}
