export function attachTakeScreenshot(
  player,
  video,
  btnEl,
  videoId,
  anchorTime,
) {
  btnEl.addEventListener('click', () => {
    if (!video.videoWidth || !video.videoHeight) return;

    player.pause();

    const canvas = document.createElement('canvas');
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    canvas.getContext('2d').drawImage(video, 0, 0);

    canvas.toBlob((blob) => {
      if (!blob) return;

      const url = URL.createObjectURL(blob);

      const totalSeconds = anchorTime + player.time();
      const isoString = new Date(totalSeconds * 1000).toISOString();
      const timestamp = isoString.split('.')[0].replace(/[-:]/g, '');

      const a = document.createElement('a');
      a.href = url;
      a.download = `Screenshot_${videoId}_${timestamp}Z.png`;

      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);

      setTimeout(() => URL.revokeObjectURL(url), 0);
    }, 'image/png');
  });
}
