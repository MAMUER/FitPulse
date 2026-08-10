import { use2FA } from './use2FA';
import './Profile.css';

export default function TwoFASetup() {
  const {
    loading,
    enabled,
    qrCode,
    secret,
    backupCodes,
    setupCode,
    setupError,
    setupSuccess,
    disableCode,
    disableError,
    panelVisible,
    setSetupCode,
    setDisableCode,
    setPanelVisible,
    handleEnable,
    handleConfirmSetup,
    handleDisable,
  } = use2FA();

  if (loading) return <div>Загрузка статуса 2FA...</div>;

  return (
    <div className='twofa-section'>
      <p
        id='twoFAStatus'
        style={{
          fontSize: 13,
          color: 'var(--text-secondary)',
          marginBottom: 12,
        }}
      >
        {enabled
          ? `Включена. Осталось резервных кодов: ${status.backup_codes_remaining}`
          : 'Не включена'}
      </p>
      <div
        style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}
      >
        {!enabled && (
          <button
            type='button'
            className='btn-secondary'
            onClick={handleEnable}
          >
            Включить 2FA
          </button>
        )}
        {enabled && (
          <button
            type='button'
            className='btn-danger'
            style={{
              color: 'var(--accent)',
              border: '1px solid rgba(255,55,95,0.4)',
            }}
            onClick={() => setPanelVisible(true)}
          >
            Отключить 2FA
          </button>
        )}
      </div>

      {panelVisible && !enabled && (
        <div
          id='totpSetupPanel'
          style={{
            padding: 12,
            background: 'var(--bg-surface)',
            borderRadius: 'var(--radius-sm)',
          }}
        >
          <p style={{ fontSize: 13, margin: '0 0 12px' }}>
            Отсканируйте QR-код в приложении-аутентификаторе.
          </p>
          {qrCode && (
            <img
              id='totpQRCode'
              src={qrCode}
              alt='TOTP QR Code'
              style={{
                width: 220,
                height: 220,
                display: 'block',
                margin: '0 auto 12px',
                background: '#fff',
                borderRadius: 'var(--radius-sm)',
              }}
            />
          )}
          <p style={{ fontSize: 13 }}>
            Не получается отсканировать? Введите секрет вручную:
          </p>
          <code id='totpManualSecret' style={{ wordBreak: 'break-all' }}>
            {secret}
          </code>
          <h4 style={{ margin: '16px 0 8px' }}>Резервные коды</h4>
          <ul
            id='totpBackupCodes'
            style={{ columns: 2, fontSize: 13, margin: '0 0 12px 20px' }}
          >
            {backupCodes.map((code) => (
              <li key={code}>
                <code>{code}</code>
              </li>
            ))}
          </ul>
          <div className='field'>
            <label htmlFor='setupCode'>Код из приложения</label>
            <input
              type='text'
              id='setupCode'
              placeholder='6-значный код'
              maxLength={6}
              inputMode='numeric'
              value={setupCode}
              onChange={(e) => setSetupCode(e.target.value)}
            />
            <div className='field-error'>{setupError}</div>
          </div>
          <div className={`auth-success ${setupSuccess ? '' : 'hidden'}`}>
            {setupSuccess}
          </div>
          <button
            type='button'
            className='btn-primary'
            onClick={handleConfirmSetup}
            style={{ marginTop: 12 }}
          >
            Подтвердить и включить 2FA
          </button>
        </div>
      )}

      {enabled && (
        <div id='disable2FAPanel' style={{ marginTop: 12 }}>
          <input
            type='text'
            placeholder='Текущий код 2FA'
            maxLength={6}
            inputMode='numeric'
            value={disableCode}
            onChange={(e) => setDisableCode(e.target.value)}
            style={{
              width: '100%',
              padding: 10,
              background: 'var(--bg-input)',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              color: 'var(--text-primary)',
            }}
          />
          <div className={`field-error ${disableError ? '' : 'hidden'}`}>
            {disableError}
          </div>
          <button
            type='button'
            className='btn-secondary'
            onClick={handleDisable}
            style={{ marginTop: 8 }}
          >
            Отключить 2FA
          </button>
        </div>
      )}
    </div>
  );
}
