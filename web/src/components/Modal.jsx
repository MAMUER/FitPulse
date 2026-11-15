import { useEffect, useRef } from 'react';

/**
 * Accessible modal built on the native <dialog> element.
 *
 * The backdrop is a real <button> so that clicking outside the panel is a
 * native interaction rather than a click handler on a non-interactive <div>
 * (which is unreachable by keyboard and invisible to assistive tech).
 *
 * The panel itself is the <dialog>, so it gets the implicit "dialog" role from
 * HTML instead of a hand-written role="dialog".
 */
export default function Modal({ label, onClose, className, children }) {
  const dialogRef = useRef(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return undefined;

    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose();
    };

    dialog.addEventListener('keydown', onKeyDown);
    return () => dialog.removeEventListener('keydown', onKeyDown);
  }, [onClose]);

  return (
    <div className={className}>
      <button
        type='button'
        aria-hidden='true'
        tabIndex={-1}
        className='fp-modal-backdrop'
        onClick={onClose}
      />
      <dialog ref={dialogRef} open aria-label={label} className='fp-dialog'>
        {children}
      </dialog>
    </div>
  );
}
