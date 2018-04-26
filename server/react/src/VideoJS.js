import React from 'react';
import videojs from 'video.js'
import axios from 'axios'
import Clappr from 'clappr'
import ChromecastPlugin from 'clappr-chromecast-plugin'

class VideoPlayer extends React.Component {

  componentDidUpdate() {
    //this.player.src(this.props.sources)
    //this.player.play()
    //if(this.props.playtime != 0){
	  //  this.player.currentTime(this.props.playtime)
    //}
    this.player = new Clappr.Player({
      parent: this.refs.player,
      source: this.props.sources[0]['src'],
      plugins: [ChromecastPlugin],
      autoplay: true,
      chromecast: {},
      hlsjsConfig: {
        enableWorker: true
      }
    })
  }

  componentDidMount() {
    //this.player = videojs(this.videoNode, this.props, function onPlayerReady() {
    //  console.log("videojs loaded")
    //});
    //this.player.on("timeupdate", (e) => {
	  //  let playtime = e.target.player.currentTime()
	  //  if((Math.round(playtime % 5)) === 0){
		//    axios.post(`${this.props.serverAddress}api/v1/state`, {playtime: Math.floor(playtime), filename: e.target.player.currentSource().name}).then(response => {
		//	    console.log("Successful state update")
		//    }).catch(error => {
		//	    console.log("Error pushing state", error);
		//    })
	  //  }
    //})

  }

  // destroy player on unmount
  componentWillUnmount() {
    if (this.player) {
      this.player.destroy()
    }
  }

  // wrap the player in a div with a `data-vjs-player` attribute
  // so videojs won't create additional wrapper in the DOM
  // see https://github.com/videojs/video.js/pull/3856
  render() {
    return (
      <div>
        <div ref="player">
        </div>
      </div>
    )
  }
}

export default VideoPlayer
