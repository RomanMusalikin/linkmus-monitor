import { useState, useEffect } from 'react';
import { Bell, Mail, Save, Send, Cpu, MemoryStick, MessageCircle, Hash } from 'lucide-react';
import { getAlertSettings, saveAlertSettings, sendTestEmail, sendTestTelegram } from '../lib/api';

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
        focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-all ${className}`}
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
];

const defaultSettings = {
  smtpHost: '', smtpPort: 587, smtpUser: '', smtpPass: '',
  fromEmail: '', toEmail: '',
  cpuThreshold: 0, ramThreshold: 0, cooldownMin: 30, enabled: false,
  tgBotToken: '', tgChatID: '', tgTopicID: 0, tgEnabled: false,
};

export default function Settings() {
  const [tab, setTab] = useState('notifications');
  const [s, setS] = useState(defaultSettings);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testingTg, setTestingTg] = useState(false);
  const [saveMsg, setSaveMsg] = useState('');
  const [testMsg, setTestMsg] = useState('');
  const [testTgMsg, setTestTgMsg] = useState('');

  useEffect(() => {
    getAlertSettings()
      .then(data => setS({ ...defaultSettings, ...data }))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const upd = field => e =>
    setS(prev => ({ ...prev, [field]: e.target.type === 'checkbox' ? e.target.checked : e.target.value }));

  const updNum = (field, min = 0, max = Infinity) => e => {
    const raw = e.target.value.replace(/[^0-9]/g, '');
    if (raw === '') { setS(prev => ({ ...prev, [field]: '' })); return; }
    setS(prev => ({ ...prev, [field]: Math.min(max, Math.max(min, parseInt(raw, 10))) }));
  };

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
