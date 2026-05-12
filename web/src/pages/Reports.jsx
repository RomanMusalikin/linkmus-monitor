import { useState, useContext } from 'react';
import { FileText, Bot, Clock, RefreshCw, CheckSquare, Square, ChevronDown, ChevronUp, AlertCircle, CalendarRange } from 'lucide-react';
import { NodesContext } from '../context/NodesContext';
import { generateReport } from '../lib/api';

const PERIODS = [
  { value: '1h',  label: 'Последний час' },
  { value: '6h',  label: 'Последние 6 часов' },
  { value: '12h', label: 'Последние 12 часов' },
  { value: '24h', label: 'Последние 24 часа' },
  { value: '7d',  label: 'Последние 7 дней' },
  { value: '30d', label: 'Последние 30 дней' },
  { value: 'custom', label: 'Произвольный период' },
];

function nowStr() {
  const d = new Date();
  return d.getFullYear() + '-'
    + String(d.getMonth() + 1).padStart(2, '0') + '-'
    + String(d.getDate()).padStart(2, '0') + 'T'
    + String(d.getHours()).padStart(2, '0') + ':'
    + String(d.getMinutes()).padStart(2, '0');
}
function daysAgoStr(n) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.getFullYear() + '-'
    + String(d.getMonth() + 1).padStart(2, '0') + '-'
    + String(d.getDate()).padStart(2, '0') + 'T'
    + String(d.getHours()).padStart(2, '0') + ':'
    + String(d.getMinutes()).padStart(2, '0');
}

function NodeCheckbox({ node, checked, onToggle }) {
  return (
    <button
      type="button"
      onClick={() => onToggle(node.name)}
      className={`flex items-center gap-3 w-full px-3 py-2.5 rounded-xl border transition-all text-left
        ${checked
          ? 'bg-violet-500/10 border-violet-500/40 text-violet-200'
          : 'bg-slate-900/40 border-slate-700/40 text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
        }`}
    >
      {checked
        ? <CheckSquare className="w-4 h-4 text-violet-400 flex-shrink-0" />
        : <Square className="w-4 h-4 flex-shrink-0" />
      }
      <div className="flex items-center gap-2 min-w-0 flex-1">
        <div className={`w-2 h-2 rounded-full flex-shrink-0 ${node.online ? 'bg-green-500' : 'bg-red-500'}`} />
        <span className="text-sm font-medium truncate">{node.displayName || node.name}</span>
        {node.displayName && node.displayName !== node.name && (
          <span className="text-xs text-slate-600 truncate hidden sm:block">({node.name})</span>
        )}
      </div>
      <span className="text-xs text-slate-600 flex-shrink-0">{node.os || ''}</span>
    </button>
  );
}

// Простой рендер markdown-подобного текста от GigaChat
function ReportText({ text }) {
  const lines = text.split('\n');
  return (
    <div className="space-y-1 text-sm text-slate-300 leading-relaxed">
      {lines.map((line, i) => {
        if (/^#{1,3}\s/.test(line)) {
          const content = line.replace(/^#+\s/, '');
          return <p key={i} className="font-semibold text-slate-100 mt-4 first:mt-0 text-base">{content}</p>;
        }
        if (/^\*\*(.+)\*\*$/.test(line)) {
          return <p key={i} className="font-semibold text-slate-200">{line.replace(/\*\*/g, '')}</p>;
        }
        if (/^[-*•]\s/.test(line)) {
          return (
            <div key={i} className="flex gap-2">
              <span className="text-violet-400 flex-shrink-0 mt-0.5">•</span>
              <span>{line.replace(/^[-*•]\s/, '').replace(/\*\*(.+?)\*\*/g, '$1')}</span>
            </div>
          );
        }
        if (line.trim() === '') return <div key={i} className="h-2" />;
        // Inline bold
        const parts = line.split(/\*\*(.+?)\*\*/g);
        return (
          <p key={i}>
            {parts.map((part, j) =>
              j % 2 === 1
                ? <strong key={j} className="text-slate-200 font-semibold">{part}</strong>
                : part
            )}
          </p>
        );
      })}
    </div>
  );
}

export default function Reports() {
  const { data: nodes } = useContext(NodesContext);
  const [selected, setSelected] = useState([]);
  const [period, setPeriod] = useState('24h');
  const [dateFrom, setDateFrom] = useState(daysAgoStr(7));
  const [dateTo, setDateTo] = useState(nowStr());
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState('');
  const [error, setError] = useState('');
  const [showAll, setShowAll] = useState(false);

  const allNodes = nodes ?? [];
  const displayedNodes = showAll ? allNodes : allNodes.slice(0, 8);

  function toggleNode(name) {
    setSelected(prev =>
      prev.includes(name) ? prev.filter(n => n !== name) : [...prev, name]
    );
  }

  function toggleAll() {
    if (selected.length === allNodes.length) {
      setSelected([]);
    } else {
      setSelected(allNodes.map(n => n.name));
    }
  }

  async function handleGenerate() {
    if (selected.length === 0) return;
    if (period === 'custom' && (!dateFrom || !dateTo)) return;
    setLoading(true);
    setError('');
    setReport('');
    try {
      const from = period === 'custom' ? dateFrom : undefined;
      const to   = period === 'custom' ? dateTo   : undefined;
      const data = await generateReport(selected, period, from, to);
      setReport(data.report || '');
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  const allSelected = allNodes.length > 0 && selected.length === allNodes.length;

  return (
    <div className="p-4 sm:p-6 max-w-4xl mx-auto">
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-1">
          <Bot className="w-6 h-6 text-violet-400" />
          <h1 className="text-2xl font-bold text-slate-100">AI-отчёты</h1>
        </div>
        <p className="text-slate-500 text-sm">Генерация отчётов по узлам с помощью GigaChat</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Левая колонка — настройки */}
        <div className="space-y-5">
          {/* Период */}
          <div className="bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-slate-700/40">
              <Clock className="w-4 h-4 text-blue-400" />
              <span className="text-sm font-semibold text-slate-200">Период анализа</span>
            </div>
            <div className="grid grid-cols-2 gap-2">
              {PERIODS.map(p => (
                <button
                  key={p.value}
                  type="button"
                  onClick={() => setPeriod(p.value)}
                  className={`px-3 py-2 rounded-lg text-sm font-medium transition-all border
                    ${p.value === 'custom' ? 'col-span-2' : ''}
                    ${period === p.value
                      ? p.value === 'custom'
                        ? 'bg-violet-500/20 border-violet-500/40 text-violet-300'
                        : 'bg-blue-500/20 border-blue-500/40 text-blue-300'
                      : 'bg-slate-900/40 border-slate-700/30 text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
                    }`}
                >
                  {p.value === 'custom'
                    ? <span className="flex items-center justify-center gap-2"><CalendarRange className="w-4 h-4" />{p.label}</span>
                    : p.label
                  }
                </button>
              ))}
            </div>

            {period === 'custom' && (
              <div className="mt-3 grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-slate-400 font-medium mb-1 uppercase tracking-wider">С</label>
                  <input
                    type="datetime-local"
                    value={dateFrom}
                    max={dateTo}
                    onChange={e => setDateFrom(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-700 text-slate-200 text-sm
                      outline-none focus:border-violet-500/60 focus:ring-1 focus:ring-violet-500/20 transition-colors
                      [color-scheme:dark]"
                  />
                </div>
                <div>
                  <label className="block text-xs text-slate-400 font-medium mb-1 uppercase tracking-wider">По</label>
                  <input
                    type="datetime-local"
                    value={dateTo}
                    min={dateFrom}
                    max={nowStr()}
                    onChange={e => setDateTo(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-700 text-slate-200 text-sm
                      outline-none focus:border-violet-500/60 focus:ring-1 focus:ring-violet-500/20 transition-colors
                      [color-scheme:dark]"
                  />
                </div>
              </div>
            )}
          </div>

          {/* Узлы */}
          <div className="bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5">
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-700/40">
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-emerald-400" />
                <span className="text-sm font-semibold text-slate-200">Узлы</span>
                {selected.length > 0 && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                    {selected.length}
                  </span>
                )}
              </div>
              <button
                type="button"
                onClick={toggleAll}
                className="text-xs text-slate-400 hover:text-slate-200 transition-colors"
              >
                {allSelected ? 'Снять всё' : 'Выбрать всё'}
              </button>
            </div>

            {allNodes.length === 0 ? (
              <p className="text-slate-500 text-sm text-center py-4">Нет узлов в системе</p>
            ) : (
              <>
                <div className="space-y-2">
                  {displayedNodes.map(node => (
                    <NodeCheckbox
                      key={node.name}
                      node={node}
                      checked={selected.includes(node.name)}
                      onToggle={toggleNode}
                    />
                  ))}
                </div>
                {allNodes.length > 8 && (
                  <button
                    type="button"
                    onClick={() => setShowAll(!showAll)}
                    className="mt-3 flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
                  >
                    {showAll
                      ? <><ChevronUp className="w-3 h-3" /> Свернуть</>
                      : <><ChevronDown className="w-3 h-3" /> Показать все ({allNodes.length})</>
                    }
                  </button>
                )}
              </>
            )}
          </div>

          {/* Кнопка */}
          <button
            type="button"
            onClick={handleGenerate}
            disabled={loading || selected.length === 0 || (period === 'custom' && (!dateFrom || !dateTo))}
            className="w-full flex items-center justify-center gap-2 px-5 py-3 rounded-xl
              bg-violet-500/20 text-violet-300 border border-violet-500/30
              hover:bg-violet-500/30 hover:text-violet-200 transition-all font-medium
              disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading
              ? <><RefreshCw className="w-4 h-4 animate-spin" /> Генерация отчёта...</>
              : <><Bot className="w-4 h-4" /> Сгенерировать отчёт</>
            }
          </button>
          {selected.length === 0 && (
            <p className="text-xs text-slate-600 text-center -mt-3">Выберите хотя бы один узел</p>
          )}
          {period === 'custom' && selected.length > 0 && (!dateFrom || !dateTo) && (
            <p className="text-xs text-slate-600 text-center -mt-3">Укажите начало и конец диапазона</p>
          )}
        </div>

        {/* Правая колонка — результат */}
        <div className="bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5 flex flex-col min-h-[400px]">
          <div className="flex items-center gap-2 mb-4 pb-3 border-b border-slate-700/40">
            <Bot className="w-4 h-4 text-violet-400" />
            <span className="text-sm font-semibold text-slate-200">Отчёт GigaChat</span>
          </div>

          {loading && (
            <div className="flex-1 flex flex-col items-center justify-center gap-3 text-slate-500">
              <RefreshCw className="w-8 h-8 animate-spin text-violet-400/60" />
              <p className="text-sm">GigaChat анализирует данные...</p>
              <p className="text-xs text-slate-600">Обычно занимает 5–15 секунд</p>
            </div>
          )}

          {error && !loading && (
            <div className="flex-1 flex flex-col items-start gap-3">
              <div className="flex items-start gap-2 p-3 rounded-xl bg-red-500/10 border border-red-500/20 w-full">
                <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
                <p className="text-sm text-red-300">{error}</p>
              </div>
              {error.includes('не настроен') && (
                <p className="text-xs text-slate-500">
                  Перейдите в <span className="text-violet-400">Настройки → GigaChat AI</span> и укажите Authorization Key из личного кабинета Sber Developers.
                </p>
              )}
            </div>
          )}

          {!loading && !error && !report && (
            <div className="flex-1 flex flex-col items-center justify-center gap-2 text-slate-600">
              <Bot className="w-10 h-10 opacity-30" />
              <p className="text-sm">Выберите узлы и период, затем нажмите «Сгенерировать отчёт»</p>
            </div>
          )}

          {!loading && !error && report && (
            <div className="flex-1 overflow-y-auto">
              <ReportText text={report} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
