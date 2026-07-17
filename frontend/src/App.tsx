import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Navbar from './components/Navbar'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import MonitorDetailPage from './pages/MonitorDetailPage'
import SettingsPage from './pages/SettingsPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token') || localStorage.getItem('apiKey')
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50">
        <Navbar />
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/monitors/:id"
            element={
              <ProtectedRoute>
                <MonitorDetailPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings"
            element={
              <ProtectedRoute>
                <SettingsPage />
              </ProtectedRoute>
            }
          />
        </Routes>
      </div>
    </BrowserRouter>
  )
}
