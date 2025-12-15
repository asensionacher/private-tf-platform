import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import ModulesPage from './pages/ModulesPage'
import ModuleDetailPage from './pages/ModuleDetailPage'
import ProvidersPage from './pages/ProvidersPage'
import ProviderDetailPage from './pages/ProviderDetailPage'
import NamespacesPage from './pages/NamespacesPage'
import NamespaceDetailPage from './pages/NamespaceDetailPage'
import DeploymentsPage from './pages/DeploymentsPage'
import DeploymentDetailPage from './pages/DeploymentDetailPage'
import SettingsPage from './pages/SettingsPage'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/modules" replace />} />
        <Route path="/modules" element={<ModulesPage />} />
        <Route path="/modules/:id" element={<ModuleDetailPage />} />
        <Route path="/providers" element={<ProvidersPage />} />
        <Route path="/providers/:id" element={<ProviderDetailPage />} />
        <Route path="/namespaces" element={<NamespacesPage />} />
        <Route path="/namespaces/:id" element={<NamespaceDetailPage />} />
        <Route path="/deployments" element={<DeploymentsPage />} />
        <Route path="/deployments/:id" element={<DeploymentDetailPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </Layout>
  )
}

export default App
