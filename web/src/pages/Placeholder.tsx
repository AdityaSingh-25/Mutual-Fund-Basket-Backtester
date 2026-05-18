import styles from './Placeholder.module.css'

interface Props {
  title: string
  note: string
}

// Placeholder is a stand-in for pages that arrive in later phases.
export default function Placeholder({ title, note }: Props) {
  return (
    <section>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.note}>{note}</p>
    </section>
  )
}
