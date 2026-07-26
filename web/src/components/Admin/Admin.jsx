import { useState, useEffect } from 'react';
import { listInvites, createInvite, revokeInvite, listUsers } from '../../utils/api';
import { useAuth } from '../../contexts/AuthContext';
import './Admin.css';

export default function Admin() {
  const { isAdmin } = useAuth();
  const [invites, setInvites] = useState([]);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [inviteForm, setInviteForm] = useState({ role: 'client', maxUses: 1 });

  useEffect(() => {
    if (isAdmin) {
      loadAdminData();
    }
  }, [isAdmin]);

  const loadAdminData = async () => {
    try {
      const [invitesData, usersData] = await Promise.allSettled([
        listInvites(),
        listUsers(),
      ]);
      if (invitesData.status === 'fulfilled') setInvites(invitesData.value || []);
      if (usersData.status === 'fulfilled') setUsers(usersData.value || []);
    } catch (e) {
      console.error('Failed to load admin data:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateInvite = async (e) => {
    e.preventDefault();
    try {
      await createInvite(inviteForm.role, '', inviteForm.maxUses);
      alert('Приглашение создано');
      setInviteForm({ role: 'client', maxUses: 1 });
      loadAdminData();
    } catch (e) {
      alert('Ошибка: ' + e.message);
    }
  };

  const handleRevoke = async (code) => {
    if (!window.confirm('Отозвать приглашение?')) return;
    try {
      await revokeInvite(code);
      alert('Приглашение отозвано');
      loadAdminData();
    } catch (e) {
      alert('Ошибка: ' + e.message);
    }
  };

  const copyToClipboard = async (text) => {
    const url = `${window.location.origin}/register/invite?code=${text}`;
    try {
      await navigator.clipboard.writeText(url);
      alert('Ссылка скопирована');
    } catch {
      const ta = document.createElement('textarea');
      ta.value = url;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
      alert('Ссылка скопирована');
    }
  };

  if (!isAdmin) {
    return <div className="view active"><div className="empty-state"><div className="empty-icon">🔒</div><h3>Доступ запрещён</h3></div></div>;
  }

  if (loading) return <div className="loading">Загрузка...</div>;

  return (
    <div className="view active">
      <div className="admin-form">
        <h3>Создать приглашение</h3>
        <form onSubmit={handleCreateInvite}>
          <div className="form-group">
            <label>Роль</label>
            <select value={inviteForm.role} onChange={e => setInviteForm(f => ({ ...f, role: e.target.value }))}>
              <option value="client">Клиент</option>
              <option value="admin">Админ</option>
            </select>
          </div>
          <div className="form-group">
            <label>Максимум использований</label>
            <input type="number" min="1" value={inviteForm.maxUses} onChange={e => setInviteForm(f => ({ ...f, maxUses: Number(e.target.value) }))} />
          </div>
          <button type="submit" className="btn-primary">Создать</button>
        </form>
      </div>

      <section className="admin-form">
        <h3>Приглашения</h3>
        <div id="invitesList" className="health-list">
          {invites.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Нет приглашений</p>
          ) : (
            invites.map(inv => (
              <div key={inv.invite_id || inv.code} className="invite-card">
                <div className="invite-header">
                  <div className="invite-code">{inv.code}</div>
                  <span className="badge">{inv.is_active !== false ? 'Активно' : 'Отозвано'}</span>
                </div>
                <div className="invite-meta">
                  Роль: {inv.role || 'client'} · Использовано: {inv.used_count || 0}/{inv.max_uses || 1}
                </div>
                <div className="invite-actions">
                  <button className="btn-secondary" onClick={() => copyToClipboard(inv.code)} style={{ padding: '8px 12px', fontSize: 13 }}>Скопировать ссылку</button>
                  {inv.is_active !== false && (
                    <button className="btn-danger-ghost" onClick={() => handleRevoke(inv.code)}>Отозвать</button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </section>

      <section className="admin-form">
        <h3>Пользователи</h3>
        <div id="usersList" className="health-list">
          {users.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', fontSize: 14 }}>Нет пользователей</p>
          ) : (
            users.map(u => (
              <div key={u.user_id || u.id} className="user-card">
                <div className="user-header">
                  <div className="user-name">{u.full_name || u.nickname || '—'}</div>
                  <span className="badge">{u.role || 'client'}</span>
                </div>
                <div className="user-email">{u.email}</div>
                <div className="user-meta">
                  Создан: {u.created_at ? new Date(u.created_at).toLocaleString('ru-RU') : '—'} · Обновлён: {u.updated_at ? new Date(u.updated_at).toLocaleString('ru-RU') : '—'}
                </div>
              </div>
            ))
          )}
        </div>
      </section>
    </div>
  );
}
