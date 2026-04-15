import { useState } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import NodeDetail from './pages/NodeDetail';
import Header from './components/layout/Header';
import Sidebar from './components/layout/Sidebar';
import { useNodes } from './hooks/useNodes';

function App() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const toggleSidebar = () => setIsSidebarOpen(!isSidebarOpen);

  // Единственный источник данных об узлах — передаём и в Header, и в Sidebar
  const { data: nodes } = useNodes();

  const onlineCount = nodes?.filter(n => n.online).length ?? 0;
  const totalCount = nodes?.length ?? 0;

  return (
    <Router>
      <div className="flex h-screen bg-slate-900 text-slate-100 overflow-hidden relative">

        <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} toggleSidebar={toggleSidebar} nodes={nodes ?? []} />

        <div className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">

          <Header toggleSidebar={toggleSidebar} onlineCount={onlineCount} totalCount={totalCount} />
          
          {/* Основной контент */}
          <main className="flex-1 overflow-x-hidden overflow-y-auto">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/node/:nodeId" element={<NodeDetail />} />
            </Routes>
          </main>

        </div>
        
        {/* Затемнение фона на мобилках при открытом меню (чтобы закрыть по клику мимо) */}
        {isSidebarOpen && (
          <div 
            className="fixed inset-0 bg-black/50 z-20 md:hidden"
            onClick={() => setIsSidebarOpen(false)}
          />
        )}

      </div>
    </Router>
  );
}

export default App;