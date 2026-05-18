import { Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import Home from './pages/Home'
import Funds from './pages/Funds'
import Baskets from './pages/Baskets'
import Backtest from './pages/Backtest'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Home />} />
        <Route path="funds" element={<Funds />} />
        <Route path="baskets" element={<Baskets />} />
        <Route path="backtest" element={<Backtest />} />
      </Route>
    </Routes>
  )
}
