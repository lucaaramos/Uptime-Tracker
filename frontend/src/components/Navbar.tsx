import { Link, useNavigate } from 'react-router-dom'

export default function Navbar() {
  const navigate = useNavigate()
  const token = localStorage.getItem('token') || localStorage.getItem('apiKey')

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('apiKey')
    navigate('/login')
  }

  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link to="/" className="text-xl font-bold text-indigo-600">
              UptimeTracker
            </Link>
            {token && (
              <div className="ml-10 flex items-center space-x-4">
                <Link
                  to="/"
                  className="text-gray-700 hover:text-indigo-600 px-3 py-2 rounded-md text-sm font-medium"
                >
                  Dashboard
                </Link>
                <Link
                  to="/settings"
                  className="text-gray-700 hover:text-indigo-600 px-3 py-2 rounded-md text-sm font-medium"
                >
                  Settings
                </Link>
              </div>
            )}
          </div>
          <div className="flex items-center">
            {token ? (
              <button
                onClick={handleLogout}
                className="text-gray-500 hover:text-gray-700 px-3 py-2 text-sm font-medium"
              >
                Logout
              </button>
            ) : (
              <Link
                to="/login"
                className="text-indigo-600 hover:text-indigo-800 px-3 py-2 text-sm font-medium"
              >
                Login
              </Link>
            )}
          </div>
        </div>
      </div>
    </nav>
  )
}
