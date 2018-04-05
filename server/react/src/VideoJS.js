import React from 'react';
import videojs from 'video.js'
import videojschromecastjs from 'videojs-chromecast'
import videojscontribhls from "videojs-contrib-hls"

class VideoPlayer extends React.Component {

  componentDidUpdate() {
    this.player = videojs(this.videoNode)
    this.player.src(this.props.sources)
    this.player.play()
  }

  componentDidMount() {
    window.VIDEOJS_NO_DYNAMIC_STYLE = true
    // instantiate Video.js
    this.player = videojs(this.videoNode, this.props, function onPlayerReady() {
      console.log("videojs loaded")
    });
  }

  // destroy player on unmount
  componentWillUnmount() {
    if (this.player) {
      this.player.dispose()
    }
  }

  // wrap the player in a div with a `data-vjs-player` attribute
  // so videojs won't create additional wrapper in the DOM
  // see https://github.com/videojs/video.js/pull/3856
  render() {
    return (
      <div>
        <div data-vjs-player>
          <video ref={ node => this.videoNode = node } className="video-js"></video>
        </div>
      </div>
    )
  }
}

export default VideoPlayer
