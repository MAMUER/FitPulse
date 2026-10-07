export default function ListItem({ title, subtitle, right }) {
  return (
    <div className='list-row'>
      <div>
        {title && <div className='list-title'>{title}</div>}
        {subtitle && <div className='muted'>{subtitle}</div>}
      </div>
      {right}
    </div>
  );
}
