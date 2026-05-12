import { useState, useEffect } from 'react';
import { Bell, Mail, Save, Send, Cpu, MemoryStick, MessageCircle, Hash, Shield, Bot, Plus, Trash2 } from 'lucide-react';
import { getAlertSettings, saveAlertSettings, sendTestEmail, sendTestTelegram, getPortSettings, savePortSettings, getGigachatSettings, saveGigachatSettings, getCustomServices, createCustomService, deleteCustomService } from '../lib/api';
import { useNodesContext } from '../context/NodesContext';

function Field({ label, hint, children }) {
  return (
    <div>
      <label className="block text-xs text-slate-400 font-medium mb-1.5 uppercase tracking-wider">{label}</label>
      {children}
      {hint && <p className="text-xs text-slate-600 mt-1">{hint}</p>}
    </div>
  );
}

function Input({ className = '', ...props }) {
  return (
    <input
      className={`w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-700 text-slate-200 text-sm
        outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-colors ${className}`}
      {...props}
    />
  );
}

function Section({ title, icon: Icon, iconColor = 'text-blue-400', children }) {
  return (
    <div className="bg-slate-800/80 border border-slate-700/50 rounded-2xl p-5">
      <div className="flex items-center gap-2 mb-5 pb-4 border-b border-slate-700/40">
        <Icon className={`w-4 h-4 ${iconColor}`} />
        <span className="text-sm font-semibold text-slate-200">{title}</span>
      </div>
      {children}
    </div>
  );
}

function Toggle({ enabled, onToggle, labelOn, labelOff, hint }) {
  return (
    <div className="flex items-center gap-3 p-3 rounded-xl bg-slate-900/40 border border-slate-700/30">
      <div className="relative inline-flex items-center cursor-pointer" onClick={onToggle}>
        <div className={`w-10 h-5 rounded-full transition-colors ${enabled ? 'bg-emerald-500' : 'bg-slate-600'}`} />
        <div className={`absolute w-3.5 h-3.5 bg-white rounded-full shadow transition-transform top-[3px] ${enabled ? 'translate-x-5 left-[3px]' : 'left-[3px]'}`} />
      </div>
      <div>
        <div className={`text-sm font-medium ${enabled ? 'text-emerald-400' : 'text-slate-400'}`}>
          {enabled ? labelOn : labelOff}
        </div>
        {hint && <div className="text-xs text-slate-600">{hint}</div>}
      </div>
    </div>
  );
}

const TABS = [
  { id: 'notifications', label: 'Уведомления', icon: Bell },
  { id: 'ports', label: 'Порты сервисов', icon: Shield },
  { id: 'gigachat', label: 'GigaChat AI', icon: Bot },
];

const defaultSettings = {
  smtpHost: '', smtpPort: 587, smtpUser: '', smtpPass: '',
  fromEmail: '', toEmail: '',
  cpuThreshold: 0, ramThreshold: 0, cooldownMin: 30, enabled: false,
  tgBotToken: '', tgChatID: '', tgTopicID: 0, tgEnabled: false,
};

const defaultPortSettings = {
  sshPort: 22, rdpPort: 3389, smbPort: 445, httpPort: 80, httpsPort: 443, winrmPort: 5985,
};

export default function Settings() {
  const { refresh: refreshNodes } = useNodesContext();
  const [tab, setTab] = useState('notifications');
  const [s, setS] = useState(defaultSettings);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testingTg, setTestingTg] = useState(false);
  const [saveMsg, setSaveMsg] = useState('');
  const [testMsg, setTestMsg] = useState('');
  const [testTgMsg, setTestTgMsg] = useState('');

  const [ports, setPorts] = useState(defaultPortSettings);
  const [portsSaving, setPortsSaving] = useState(false);
  const [portsSaveMsg, setPortsSaveMsg] = useState('');

  const [gigachat, setGigachat] = useState({ clientId: '', clientSecret: '', scope: 'GIGACHAT_API_PERS' });
  const [gcSaving, setGcSaving] = useState(false);
  const [gcSaveMsg, setGcSaveMsg] = useState('');

  const [customServices, setCustomServices] = useState([]);
  const [newSvcName, setNewSvcName] = useState('');
  const [newSvcPort, setNewSvcPort] = useState('');
  const [svcAdding, setSvcAdding] = useState(false);
  const [svcMsg, setSvcMsg] = useState('');

  useEffect(() => {
    getAlertSettings()
      .then(data => setS({ ...defaultSettings, ...data }))
      .catch(() => {})
      .finally(() => setLoading(false));
    getPortSettings()
      .then(data => setPorts({ ...defaultPortSettings, ...data }))
      .catch(() => {});
    getGigachatSettings()
      .then(data => setGigachat(prev => ({ ...prev, ...data })))
      .catch(() => {});
    getCustomServices()
      .then(setCustomServices)
      .catch(() => {});
  }, []);

  async function handleAddService(e) {
    e.preventDefault();
    const port = parseInt(newSvcPort, 10);
    if (!newSvcName.trim() || !port || port < 1 || port > 65535) return;
    setSvcAdding(true); setSvcMsg('');
    try {
      const svc = await createCustomService(newSvcName.trim(), port);
      setCustomServices(prev => [...prev, svc]);
      setNewSvcName('');
      setNewSvcPort('');
      refreshNodes?.();
    } catch (err) {
      setSvcMsg('Ошибка: ' + err.message);
    } finally {
      setSvcAdding(false);
      setTimeout(() => setSvcMsg(''), 3000);
    }
  }

  async function handleDeleteService(id) {
    try {
      await deleteCustomService(id);
      setCustomServices(prev => prev.filter(s => s.id !== id));
      refreshNodes?.();
    } catch (err) {
      setSvcMsg('Ошибка удаления: ' + err.message);
      setTimeout(() => setSvcMsg(''), 3000);
    }
  }

  const upd = field => e =>
    setS(prev => ({ ...prev, [field]: e.target.type === 'checkbox' ? e.target.checked : e.target.value }));

  const updNum = (field, min = 0, max = Infinity) => e => {
    const raw = e.target.value.replace(/[^0-9]/g, '');
    if (raw === '') { setS(prev => ({ ...prev, [field]: '' })); return; }
    setS(prev => ({ ...prev, [field]: Math.min(max, Math.max(min, parseInt(raw, 10))) }));
  };

  const updPort = field => e => {
    const raw = e.target.value.replace(/[^0-9]/g, '');
    if (raw === '') { setPorts(prev => ({ ...prev, [field]: '' })); return; }
    setPorts(prev => ({ ...prev, [field]: Math.min(65535, Math.max(1, parseInt(raw, 10))) }));
  };

  async function handlePortsSave(e) {
    e.preventDefault();
    setPortsSaving(true); setPortsSaveMsg('');
    try {
      await savePortSettings(ports);
      setPortsSaveMsg('✓ Сохранено');
      setTimeout(() => setPortsSaveMsg(''), 2500);
    } catch (err) {
      setPortsSaveMsg('Ошибка: ' + err.message);
    } finally {
      setPortsSaving(false);
    }
  }

  async function handleSave(e) {
    e.preventDefault();
    setSaving(true); setSaveMsg('');
    try {
      await saveAlertSettings(s);
      setSaveMsg('✓ Сохранено');
      setTimeout(() => setSaveMsg(''), 2500);
    } catch (err) {
      setSaveMsg('Ошибка: ' + err.message);
    } finally {
      setSaving(false);
    }
  }

  async function handleGcSave(e) {
    e.preventDefault();
    setGcSaving(true); setGcSaveMsg('');
    try {
      await saveGigachatSettings(gigachat);
      setGcSaveMsg('✓ Сохранено');
      setTimeout(() => setGcSaveMsg(''), 2500);
    } catch (err) {
      setGcSaveMsg('Ошибка: ' + err.message);
    } finally {
      setGcSaving(false);
    }
  }

  async function handleTestEmail() {
    setTesting(true); setTestMsg('');
    try {
      await sendTestEmail();
      setTestMsg('✓ Письмо отправлено — проверьте почту');
    } catch (err) {
      setTestMsg('Ошибка: ' + err.message);
    } finally {
      setTesting(false);
      setTimeout(() => setTestMsg(''), 5000);
    }
  }

  async function handleTestTg() {
    setTestingTg(true); setTestTgMsg('');
    try {
      await sendTestTelegram();
      setTestTgMsg('✓ Сообщение отправлено');
    } catch (err) {
      setTestTgMsg('Ошибка: ' + err.message);
    } finally {
      setTestingTg(false);
      setTimeout(() => setTestTgMsg(''), 5000);
    }
  }

  if (loading) return <div className="p-6 text-slate-400 text-sm">Загрузка настроек...</div>;

  return (
    <div className="p-4 sm:p-6 max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-100">Настройки</h1>
        <p className="text-slate-500 text-sm mt-1">Конфигурация системы мониторинга</p>
      </div>

      {/* Вкладки */}
      <div className="flex gap-1 mb-6 bg-slate-800/60 border border-slate-700/50 rounded-xl p-1">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === t.id
                ? 'bg-slate-700 text-slate-100'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <t.icon className="w-4 h-4" />
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'ports' && (
        <form onSubmit={handlePortsSave} className="space-y-5">
          <Section title="Порты сервисов" icon={Shield} iconColor="text-emerald-400">
            <p className="text-xs text-slate-500 mb-4">
              Порты используются для TCP-проб в карточке узла. Измените, если сервисы работают на нестандартных портах.
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Field label="SSH" hint="По умолчанию: 22">
                <Input type="text" inputMode="numeric" value={ports.sshPort}
                  onChange={updPort('sshPort')} placeholder="22" />
              </Field>
              <Field label="HTTP" hint="По умолчанию: 80">
                <Input type="text" inputMode="numeric" value={ports.httpPort}
                  onChange={updPort('httpPort')} placeholder="80" />
              </Field>
              <Field label="HTTPS" hint="По умолчанию: 443">
                <Input type="text" inputMode="numeric" value={ports.httpsPort}
                  onChange={updPort('httpsPort')} placeholder="443" />
              </Field>
              <Field label="Remote Desktop (RDP)" hint="По умолчанию: 3389">
                <Input type="text" inputMode="numeric" value={ports.rdpPort}
                  onChange={updPort('rdpPort')} placeholder="3389" />
              </Field>
              <Field label="File Sharing (SMB)" hint="По умолчанию: 445">
                <Input type="text" inputMode="numeric" value={ports.smbPort}
                  onChange={updPort('smbPort')} placeholder="445" />
              </Field>
              <Field label="WinRM" hint="По умолчанию: 5985">
                <Input type="text" inputMode="numeric" value={ports.winrmPort}
                  onChange={updPort('winrmPort')} placeholder="5985" />
              </Field>
            </div>
          </Section>

          <div className="flex items-center gap-4">
            <button type="submit" disabled={portsSaving}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-blue-500/20 text-blue-400 border border-blue-500/30
                hover:bg-blue-500/30 hover:text-blue-300 transition-all font-medium text-sm disabled:opacity-50">
              <Save className="w-4 h-4" />
              {portsSaving ? 'Сохранение...' : 'Сохранить'}
            </button>
            {portsSaveMsg && (
              <span className={`text-sm ${portsSaveMsg.startsWith('✓') ? 'text-emerald-400' : 'text-red-400'}`}>{portsSaveMsg}</span>
            )}
          </div>

          {/* Пользовательские сервисы */}
          <Section title="Пользовательские сервисы" icon={Plus} iconColor="text-violet-400">
            <p className="text-xs text-slate-500 mb-4">
              Добавьте свои сервисы для TCP-мониторинга. Каждый узел можно настроить отдельно — выбрать, какие сервисы отображать.
            </p>

            {customServices.length > 0 && (
              <div className="space-y-2 mb-4">
                {customServices.map(svc => (
                  <div key={svc.id} className="flex items-center justify-between px-3 py-2.5 rounded-xl bg-slate-900/50 border border-slate-700/40">
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium text-slate-200">{svc.name}</span>
                      <span className="text-xs px-2 py-0.5 rounded bg-slate-700/60 text-slate-400 font-mono">:{svc.port}</span>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleDeleteService(svc.id)}
                      className="p-1.5 rounded-lg text-slate-600 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                      title="Удалить сервис"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex gap-2">
              <div className="flex-1">
                <Input
                  type="text"
                  value={newSvcName}
                  onChange={e => setNewSvcName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleAddService(e)}
                  placeholder="Название (например, PostgreSQL)"
                  maxLength={40}
                />
              </div>
              <div className="w-24">
                <Input
                  type="text"
                  inputMode="numeric"
                  value={newSvcPort}
                  onChange={e => setNewSvcPort(e.target.value.replace(/[^0-9]/g, ''))}
                  onKeyDown={e => e.key === 'Enter' && handleAddService(e)}
                  placeholder="Порт"
                />
              </div>
              <button
                type="button"
                onClick={handleAddService}
                disabled={svcAdding || !newSvcName.trim() || !newSvcPort}
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-violet-500/20 text-violet-300 border border-violet-500/30
                  hover:bg-violet-500/30 transition-all text-sm font-medium disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
              >
                <Plus className="w-4 h-4" />
                Добавить
              </button>
            </div>
            {svcMsg && <p className="text-xs mt-2 text-red-400">{svcMsg}</p>}
          </Section>
        </form>
      )}

      {tab === 'gigachat' && (
        <form onSubmit={handleGcSave} className="space-y-5">
          <Section title="GigaChat API" icon={Bot} iconColor="text-violet-400">
            <p className="text-xs text-slate-500 mb-4">
              Учётные данные для интеграции с GigaChat. Получить ключ можно в{' '}
              <span className="text-slate-400">личном кабинете Sber Developers</span>{' '}
              → Настройка API → Authorization key.
              Используется для генерации AI-отчётов по узлам.
            </p>
            <div className="space-y-4">
              <Field label="Client ID" hint="Client ID из личного кабинета (необязательно)">
                <Input
                  type="text"
                  value={gigachat.clientId}
                  onChange={e => setGigachat(p => ({ ...p, clientId: e.target.value }))}
                  placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  autoComplete="off"
                />
              </Field>
              <Field label="Authorization Key" hint="Ключ авторизации из личного кабинета Sber Developers → Настройка API">
                <Input
                  type="password"
                  value={gigachat.clientSecret}
                  onChange={e => setGigachat(p => ({ ...p, clientSecret: e.target.value }))}
                  placeholder="••••••••••••••••"
                  autoComplete="new-password"
                />
              </Field>
              <Field label="Scope" hint="GIGACHAT_API_PERS — личный, GIGACHAT_API_CORP — корпоративный">
                <select
                  value={gigachat.scope}
                  onChange={e => setGigachat(p => ({ ...p, scope: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-700 text-slate-200 text-sm
                    outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-colors"
                >
                  <option value="GIGACHAT_API_PERS">GIGACHAT_API_PERS (личный)</option>
                  <option value="GIGACHAT_API_B2B">GIGACHAT_API_B2B (B2B)</option>
                  <option value="GIGACHAT_API_CORP">GIGACHAT_API_CORP (корпоративный)</option>
                </select>
              </Field>
            </div>
          </Section>

          <div className="flex items-center gap-4">
            <button type="submit" disabled={gcSaving}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-violet-500/20 text-violet-400 border border-violet-500/30
                hover:bg-violet-500/30 hover:text-violet-300 transition-all font-medium text-sm disabled:opacity-50">
              <Save className="w-4 h-4" />
              {gcSaving ? 'Сохранение...' : 'Сохранить'}
            </button>
            {gcSaveMsg && (
              <span className={`text-sm ${gcSaveMsg.startsWith('✓') ? 'text-emerald-400' : 'text-red-400'}`}>{gcSaveMsg}</span>
            )}
          </div>
        </form>
      )}

      {tab === 'notifications' && (
        <form onSubmit={handleSave} className="space-y-5">

          {/* ── Пороги алертов ── */}
          <Section title="Пороги уведомлений" icon={Bell} iconColor="text-amber-400">
            <Toggle
              enabled={s.enabled || s.tgEnabled}
              onToggle={() => {}}
              labelOn="Хотя бы один канал активен"
              labelOff="Все уведомления отключены"
              hint="Управляйте каналами ниже независимо"
            />
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
              <Field label="CPU — порог (%)" hint="0 = не отслеживать">
                <div className="relative">
                  <Cpu className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-500" />
                  <Input type="text" inputMode="numeric" value={s.cpuThreshold}
                    onChange={updNum('cpuThreshold', 0, 100)} placeholder="85" className="pl-8" />
                </div>
              </Field>
              <Field label="RAM — порог (%)" hint="0 = не отслеживать">
                <div className="relative">
                  <MemoryStick className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-500" />
                  <Input type="text" inputMode="numeric" value={s.ramThreshold}
                    onChange={updNum('ramThreshold', 0, 100)} placeholder="90" className="pl-8" />
                </div>
              </Field>
            </div>
            <div className="mt-4 max-w-xs">
              <Field label="Кулдаун (минуты)" hint="Минимальный интервал между повторными уведомлениями по одному узлу">
                <Input type="text" inputMode="numeric" value={s.cooldownMin}
                  onChange={updNum('cooldownMin', 1, 1440)} placeholder="30" />
              </Field>
            </div>
          </Section>

          {/* ── Email ── */}
          <Section title="Email (SMTP)" icon={Mail} iconColor="text-cyan-400">
            <Toggle
              enabled={s.enabled}
              onToggle={() => setS(p => ({ ...p, enabled: !p.enabled }))}
              labelOn="Email-уведомления включены"
              labelOff="Email-уведомления отключены"
              hint="Проверка каждые 60 секунд"
            />

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4">
              <div className="sm:col-span-2">
                <Field label="SMTP-хост" hint="smtp.gmail.com, smtp.yandex.ru">
                  <Input type="text" value={s.smtpHost} onChange={upd('smtpHost')} placeholder="smtp.gmail.com" />
                </Field>
              </div>
              <Field label="Порт">
                <Input type="number" value={s.smtpPort} onChange={updNum('smtpPort')} placeholder="587" min={1} max={65535} />
              </Field>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
              <Field label="Логин (email отправителя)">
                <Input type="text" value={s.smtpUser} onChange={upd('smtpUser')} placeholder="alerts@example.com" autoComplete="off" />
              </Field>
              <Field label="Пароль / пароль приложения" hint="Для Gmail/Яндекс — пароль приложения">
                <Input type="password" value={s.smtpPass} onChange={upd('smtpPass')} placeholder="••••••••" autoComplete="new-password" />
              </Field>
            </div>
            <div className="mt-4">
              <Field label="Получатель уведомлений">
                <Input type="email" value={s.toEmail} onChange={upd('toEmail')} placeholder="admin@example.com" />
              </Field>
            </div>
            <div className="mt-4 flex items-center gap-3">
              <button type="button" onClick={handleTestEmail} disabled={testing || !s.smtpHost || !s.smtpUser}
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-cyan-500/10 text-cyan-400 border border-cyan-500/20
                  hover:bg-cyan-500/20 transition-all text-sm font-medium disabled:opacity-40 disabled:cursor-not-allowed">
                <Send className="w-3.5 h-3.5" />
                {testing ? 'Отправка...' : 'Тестовое письмо'}
              </button>
              {testMsg && (
                <span className={`text-sm ${testMsg.startsWith('✓') ? 'text-emerald-400' : 'text-red-400'}`}>{testMsg}</span>
              )}
            </div>
          </Section>

          {/* ── Telegram ── */}
          <Section title="Telegram" icon={MessageCircle} iconColor="text-blue-400">
            <Toggle
              enabled={s.tgEnabled}
              onToggle={() => setS(p => ({ ...p, tgEnabled: !p.tgEnabled }))}
              labelOn="Telegram-уведомления включены"
              labelOff="Telegram-уведомления отключены"
              hint="Проверка каждые 60 секунд"
            />

            <div className="mt-4 space-y-4">
              <Field label="Токен бота" hint="Получить у @BotFather → /newbot">
                <Input type="password" value={s.tgBotToken} onChange={upd('tgBotToken')}
                  placeholder="1234567890:AAF..." autoComplete="new-password" />
              </Field>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <Field label="Chat ID" hint="ID чата или канала. Получить через @userinfobot">
                  <Input type="text" value={s.tgChatID} onChange={upd('tgChatID')}
                    placeholder="-100123456789" />
                </Field>
                <Field label="Topic ID (опционально)" hint="ID топика в супергруппе. 0 = не использовать">
                  <div className="relative">
                    <Hash className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-500" />
                    <Input type="text" inputMode="numeric" value={s.tgTopicID || ''}
                      onChange={updNum('tgTopicID', 0)} placeholder="0" className="pl-8" />
                  </div>
                </Field>
              </div>
            </div>

            <div className="mt-4 flex items-center gap-3">
              <button type="button" onClick={handleTestTg} disabled={testingTg || !s.tgBotToken || !s.tgChatID}
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20
                  hover:bg-blue-500/20 transition-all text-sm font-medium disabled:opacity-40 disabled:cursor-not-allowed">
                <Send className="w-3.5 h-3.5" />
                {testingTg ? 'Отправка...' : 'Тестовое сообщение'}
              </button>
              {testTgMsg && (
                <span className={`text-sm ${testTgMsg.startsWith('✓') ? 'text-emerald-400' : 'text-red-400'}`}>{testTgMsg}</span>
              )}
            </div>
          </Section>

          {/* Сохранить */}
          <div className="flex items-center gap-4">
            <button type="submit" disabled={saving}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-blue-500/20 text-blue-400 border border-blue-500/30
                hover:bg-blue-500/30 hover:text-blue-300 transition-all font-medium text-sm disabled:opacity-50">
              <Save className="w-4 h-4" />
              {saving ? 'Сохранение...' : 'Сохранить'}
            </button>
            {saveMsg && (
              <span className={`text-sm ${saveMsg.startsWith('✓') ? 'text-emerald-400' : 'text-red-400'}`}>{saveMsg}</span>
            )}
          </div>

        </form>
      )}
    </div>
  );
}
