import React from 'react';
import videojs from 'video.js'
import axios from 'axios'
//import chromecast from 'videojs-chromecast'
import videojshls from "videojs-contrib-hls"
import videojsflash from "videojs-flash"

class VideoPlayer extends React.Component {

  componentDidUpdate() {
    this.player = videojs(this.videoNode)
    this.player.src(this.props.sources)
    this.player.play()
    if(this.props.playtime != 0){
	    this.player.currentTime(this.props.playtime)
    }
  }

  componentDidMount() {
    window.VIDEOJS_NO_DYNAMIC_STYLE = true
    this.player = videojs(this.videoNode, this.props, function onPlayerReady() {
      console.log("videojs loaded")
    });
    this.player.on("timeupdate", (e) => {
	    let playtime = e.target.player.currentTime()
	    if((Math.round(playtime % 5)) === 0){
		    axios.post(`${this.props.serverAddress}api/v1/state`, {playtime: Math.floor(playtime), filename: e.target.player.currentSource().name}).then(response => {
			    console.log("Successful state update")
		    }).catch(error => {
			    console.log("Error pushing state", error);
		    })
	    }
    })

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
          <video ref={ node => this.videoNode = node } className="video-js vjs-default-skin" controls></video>
        </div>
      </div>
    )
  }
}

export default VideoPlayer
