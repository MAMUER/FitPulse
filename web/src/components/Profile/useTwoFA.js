import { useCallback, useEffect, useState } from 'react';
import {
  confirm2FA,
  disable2FA,
  get2FAStatus,
  setup2FA,
} from '../../utils/api';

export function useTwoFA() {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [qrCode, setQrCode] = useState('');
  const [secret, setSecret] = useState('');
  const [backupCodes, setBackupCodes] = useState([]);
  const [setupCode, setSetupCode] = useState('');
  const [setupError, setSetupError] = useState('');
  const [setupSuccess, setSetupSuccess] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disableError, setDisableError] = useState('');
  const [panelVisible, setPanelVisible] = useState(false);

  const loadStatus = useCallback(async () => {
    try {
      const data = await get2FAStatus();
      setStatus(data);
    } catch (e) {
      console.error('Failed to load 2FA status:', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const handleEnable = async () => {
    setPanelVisible(true);
    setSetupError('');
    setSetupSuccess('');
    try {
      const data = await setup2FA();
      setQrCode(data.qr_code_base64 || '');
      setSecret((data.secret || '').replace(/(.{4})/g, '$1 ').trim());
      setBackupCodes(data.backup_codes || []);
    } catch (err) {
      setSetupError(err.message);
    }
  };

  const handleConfirmSetup = async () => {
    setSetupError('');
    if (!/^\d{6}$/.test(setupCode)) {
      setSetupError('Введите 6-значный код');
      return;
    }
    try {
      const secretClean = secret.replace(/\s+/g, '');
      await confirm2FA(setupCode, secretClean, backupCodes);
      setSetupSuccess(
        '2FA включена. Сохраните резервные коды в надёжном месте.'
      );
      setPanelVisible(false);
      loadStatus();
    } catch (err) {
      setSetupError(err.message);
    }
  };

  const handleDisable = async () => {
    setDisableError('');
    if (!disableCode) {
      setDisableError('Введите код 2FA');
      return;
    }
    try {
      await disable2FA(disableCode);
      setDisableCode('');
      loadStatus();
    } catch (err) {
      setDisableError(err.message);
    }
  };

  const enabled = status?.enabled;

  return {
    loading,
    enabled,
    status,
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
  };
}
