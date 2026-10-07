export default function MetricCard({ value, label, suffix }) {
  return (
    <div className='metric'>
      <div className='metric-val'>
        {value}
        {suffix && <span className='muted'>{suffix}</span>}
      </div>
      <div className='metric-label'>{label}</div>
    </div>
  );
}
