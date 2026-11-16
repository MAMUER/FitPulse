/**
 * Video thumbnail card shared by the Videos screen and the Training screen.
 * Both used to render the same 19-line block inline.
 */
export default function VideoCard({ video, index, onSelect }) {
  return (
    <button
      type='button'
      className='video-card'
      style={{ backgroundImage: `url('${video.image}')` }}
      onClick={() => onSelect(index)}
    >
      <span className='play'>
        <i className='fas fa-play' />
      </span>
      <span className='video-text'>
        <strong>{video.title}</strong>
        <small>
          {video.level} · {video.duration}
        </small>
      </span>
    </button>
  );
}
