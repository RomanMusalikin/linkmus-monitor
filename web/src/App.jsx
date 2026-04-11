import { useState } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import NodeDetail from './pages/NodeDetail';
import Header from './components/layout/Header';
import Sidebar from './components/layout/Sidebar';

function App() {
  // Состояние: открыт ли сайдбар (по умолчанию true для ПК, на мобилках скроем через CSS)
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);

  // Функция для переключения состояния
  const toggleSidebar = () => setIsSidebarOpen(!isSidebarOpen);

  return (
    <Router>
      <div className="flex h-screen bg-slate-900 text-slate-100 overflow-hidden relative">
        
        {/* Сайдбар (передаем состояние и функцию закрытия для мобилок) */}
        <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} />

        {/* Правая часть (Шапка + Рабочая область) */}
        <div className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">
          
          {/* Шапка (передаем функцию переключения) */}
          <Header toggleSidebar={toggleSidebar} />
          
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