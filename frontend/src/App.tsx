import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import ModulesPage from './pages/ModulesPage'
import ModuleDetailPage from './pages/ModuleDetailPage'
import ProvidersPage from './pages/ProvidersPage'
import ProviderDetailPage from './pages/ProviderDetailPage'
import NamespacesPage from './pages/NamespacesPage'
import NamespaceDetailPage from './pages/NamespaceDetailPage'
import DeploymentTFStatePage from './pages/DeploymentTFStatePage'
import ApiKeysPage from './pages/ApiKeysPage'
import BackendPage from './pages/BackendPage'
import TFStateBrowserPage from './pages/TFStateBrowserPage'

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
        <Route path="/deployments/:id/tfstate" element={<DeploymentTFStatePage />} />
        <Route path="/api-keys" element={<ApiKeysPage />} />
        <Route path="/backend" element={<BackendPage />} />
        <Route path="/tfstate" element={<TFStateBrowserPage />} />
      </Routes>
    </Layout>
  )
}

export default App
