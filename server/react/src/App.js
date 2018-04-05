import React from 'react';
import './App.css';
import FileList from './FileList.js'
import VideoPlayer from './VideoJS.js'
import Grid from 'material-ui/Grid';
import CssBaseline from 'material-ui/CssBaseline';

class App extends React.Component {
    state = {
      videoJsOptions:{
        autoplay: false,
        controls: true,
      }
    }
  playMovie = (name) => {
    this.setState({videoJsOptions: {
        autoplay: true,
        controls: true,
        sources: [{
          src: 'http://localhost:8080/'+name+'/hls-transcoding-manifest.m3u8',
          type: 'application/x-mpegURL',
        }], chromecast:{
          appId:'2A952047'
        }}})
  }

  render() {
    return (
       <React.Fragment>
        <CssBaseline />
        <div className="App">
          <header className="App-header">
            <h1 className="App-title">Bytesized Streaming</h1>
          </header>
          <Grid container spacing={24}>
            <Grid item xs>
              <FileList onClickMethod={this.playMovie}/>
            </Grid>
            <Grid item xs>
              <VideoPlayer {...this.state.videoJsOptions} />
            </Grid>
          </Grid>
        </div>
       </React.Fragment>
    );
  }
}

export default App;
