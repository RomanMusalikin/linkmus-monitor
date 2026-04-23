import { useState } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import NodeDetail from './pages/NodeDetail';
import LoginPage from './pages/LoginPage';
import Header from './components/layout/Header';
import Sidebar from './components/layout/Sidebar';
import { useNodes } from './hooks/useNodes';
import { useAuth } from './hooks/useAuth';
import { NodesContext } from './context/NodesContext';

function AppShell({ onLogout }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const toggleSidebar = () => setIsSidebarOpen(!isSidebarOpen);
  const nodesState = useNodes();
  const { data: nodes } = nodesState;
  const onlineCount = nodes?.filter(n => n.online).length ?? 0;
  const totalCount = nodes?.length ?? 0;

  return (
    <NodesContext.Provider value={nodesState}>
    <div className="flex h-screen bg-slate-900 text-slate-100 overflow-hidden relative">
      <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} toggleSidebar={toggleSidebar} nodes={nodes ?? []} />

      <div className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">
        <Header onlineCount={onlineCount} totalCount={totalCount} onLogout={onLogout} />
        <main className="flex-1 overflow-x-hidden overflow-y-auto">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/node/:nodeId" element={<NodeDetail />} />
          </Routes>
        </main>
      </div>

      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-20 md:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}
    </div>
    </NodesContext.Provider>
  );
}

function App() {
  const { status, refresh } = useAuth();

  if (status === 'loading') {
    return (
      <div style={{
        minHeight: '100vh',
        background: '#09090f',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <div style={{ color: '#444455', fontSize: 14, letterSpacing: '0.1em' }}>
          ...
        </div>
      </div>
    );
  }

  if (status === 'setup' || status === 'login') {
    return <LoginPage mode={status} onAuth={refresh} />;
  }

  return (
    <Router>
      <AppShell onLogout={refresh} />
    </Router>
  );
}

export default App;
