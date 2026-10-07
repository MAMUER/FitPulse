export default function Panel({ title, subtitle, right, children }) {
  return (
    <div className='panel'>
      <div className='panel-head'>
        <div>
          {title && <div className='panel-title'>{title}</div>}
          {subtitle && <div className='panel-sub'>{subtitle}</div>}
        </div>
        {right}
      </div>
      <div className='panel-body'>{children}</div>
    </div>
  );
}
