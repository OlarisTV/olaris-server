const videoElement = document.createElement('video');

export function isCodecCompatible(codec) {
  let response = videoElement.canPlayType("video/mp4; codecs=" + codec)
  return response === "probably"
}