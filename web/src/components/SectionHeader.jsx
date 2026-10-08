export default function SectionHeader({ title, right }) {
  return (
    <div className='panel-head'>
      <div>{title && <div className='panel-title'>{title}</div>}</div>
      {right}
    </div>
  );
}
