import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Activity } from 'lucide-react';
import Dashboard from './pages/Dashboard';
import NodeDetail from './pages/NodeDetail';

// Простой компонент шапки прямо здесь для начала (потом вынесем в components/layout/Header.jsx)
const Header = () => (
  <header className="bg-slate-800 border-b border-slate-700 px-6 py-4 flex items-center justify-between">
    <div className="flex items-center gap-3">
      <div className="bg-blue-500/20 p-2 rounded-lg">
        <Activity className="text-blue-500 w-6 h-6" />
      </div>
      <h1 className="text-xl font-bold tracking-tight">LinkMus Monitor</h1>
    </div>
    <div className="text-slate-400 text-sm">
      {/* Здесь потом сделаем живые часы и статус */}
      Онлайн: 6/6 узлов
    </div>
  </header>
);

function App() {
  return (
    <Router>
      <div className="min-h-screen flex flex-col">
        <Header />
        
        {/* Основной контент */}
        <main className="flex-1 overflow-auto">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/node/:nodeId" element={<NodeDetail />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;