import axios from 'axios'

const TOKEN_KEY = 'xstek_auth_token'
const USER_KEY = 'xstek_auth_user'

// Flag to prevent multiple redirects
let isRedirectingToLogin = false

// Session expired handler - clears auth state and redirects to login
export function handleSessionExpired(message?: string) {
  // Prevent multiple redirects
  if (isRedirectingToLogin) return
  isRedirectingToLogin = true

  console.warn('Session expired:', message || 'Token invalid or expired')

  // Clear auth state from localStorage
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  
  // Clear axios default headers
  delete axios.defaults.headers.common['Authorization']

  // Store a message to show on login page
  sessionStorage.setItem('session_expired_message', message || 'Your session has expired. Please log in again.')

  // Redirect to login page
  // Using window.location to ensure a full page reload and reset of Vue state
  window.location.href = '/login?session_expired=true'
}

// Create axios instance with base URL for API calls
const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json'
  }
})

// Add request interceptor to include auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem(TOKEN_KEY)
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Add response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid - trigger logout and redirect
      handleSessionExpired('Your session has expired. Please log in again.')
    }
    return Promise.reject(error)
  }
)

// Helper function to wrap fetch calls with session expiry handling
export async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const token = localStorage.getItem(TOKEN_KEY)
  
  const headers = new Headers(options.headers || {})
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(url, {
    ...options,
    headers
  })

  // Check for 401 and handle session expiry
  if (response.status === 401) {
    handleSessionExpired('Your session has expired. Please log in again.')
    throw new Error('Session expired')
  }

  return response
}

export default api
