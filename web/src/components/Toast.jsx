import { useApp } from '../contexts/AppContext';

// Toast — глобальный контейнер уведомлений. notify() в AppContext пишет
// state.__toast, а этот компонент показывает его поверх всех экранов.
// Раньше state.__toast нигде не рендерился, поэтому все notify()-сообщения
// (ошибки входа, «не настроено» и т.д.) были невидимы.
// Использует существующие стили .fp-toast + .fp-toast.show из index.css.
export default function Toast() {
  const { state } = useApp();
  const message = state.__toast;
  if (!message) return null;

  return <output className='fp-toast show'>{message}</output>;
}
