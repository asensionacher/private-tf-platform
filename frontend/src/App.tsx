import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './context/AuthContext'
import Layout from './components/Layout'
import LoginPage from './pages/LoginPage'
import ModulesPage from './pages/ModulesPage'
import ModuleDetailPage from './pages/ModuleDetailPage'
import ProvidersPage from './pages/ProvidersPage'
import ProviderDetailPage from './pages/ProviderDetailPage'
import NamespacesPage from './pages/NamespacesPage'
import NamespaceDetailPage from './pages/NamespaceDetailPage'
import DeploymentTFStatePage from './pages/DeploymentTFStatePage'
import ApiKeysPage from './pages/ApiKeysPage'
import TFStateBrowserPage from './pages/TFStateBrowserPage'
import UsersPage from './pages/UsersPage'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { isAdmin } = useAuth()
  if (!isAdmin) return <Navigate to="/modules" replace />
  return <>{children}</>
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <Layout>
              <Routes>
                <Route path="/" element={<Navigate to="/modules" replace />} />
                <Route path="/modules" element={<ModulesPage />} />
                <Route path="/modules/:id" element={<ModuleDetailPage />} />
                <Route path="/providers" element={<ProvidersPage />} />
                <Route path="/providers/:id" element={<ProviderDetailPage />} />
                <Route path="/namespaces" element={<NamespacesPage />} />
                <Route path="/namespaces/:id" element={<NamespaceDetailPage />} />
                <Route path="/deployments/:id/tfstate" element={<DeploymentTFStatePage />} />
                <Route path="/api-keys" element={<ApiKeysPage />} />
                <Route path="/tfstate" element={<TFStateBrowserPage />} />
                <Route
                  path="/users"
                  element={
                    <RequireAdmin>
                      <UsersPage />
                    </RequireAdmin>
                  }
                />
              </Routes>
            </Layout>
          </RequireAuth>
        }
      />
    </Routes>
  )
}

export default App
