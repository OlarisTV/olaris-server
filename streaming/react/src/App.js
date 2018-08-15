import React from 'react';
import './App.css';
import FileList from './FileList.js'
import VideoPlayer from './VideoJS.js'
import Grid from 'material-ui/Grid';
import CssBaseline from 'material-ui/CssBaseline';

class ServerInput extends React.Component {
	state = {serverAddress: window.location.origin, connected: false}

	onChange = (event) => {
		this.setState({serverAddress: event.target.value})
	}

	onSubmit = (event) => {
	  event.preventDefault();
	  console.log("Submit server:",this.state.serverAddress)
	  this.props.onSubmitFunction(this.state.serverAddress)
	}

	render() {
		return (
			<form onSubmit={this.onSubmit}>
				<input type="text" placeholder="Server address" onChange={this.onChange} value={this.state.serverAddress} />
				<button type="submit">Login to server</button>
			</form>
		)
	}
}


class App extends React.Component {
	state = {
		serverAddress: "",
		playtime: 0,
		videoJsOptions:{
		}
	}

  updateServerAddress = (src) => {
	  this.setState({serverAddress: src, connected: true})
  }

  playMovie = (src, name, playtime) => {
    this.setState({playtime: playtime, videoJsOptions: {
      autoplay: true,
      controls: true,
      enableLowInitialPlaylist: true,
      sources: [{
        src: src,
        type: 'application/x-mpegURL',
        name: name
      }]}})
  }
  mainContainer = () => {
	  if(this.state.connected == true){
		  return (
		  <Grid container spacing={24}>
		    <Grid item xs>
		      <FileList serverAddress={this.state.serverAddress} onClickMethod={this.playMovie}/>
		    </Grid>
		    <Grid item xs>
		      <VideoPlayer playtime={this.state.playtime} serverAddress={this.state.serverAddress} {...this.state.videoJsOptions} />
		    </Grid>
		  </Grid>
		  )
	  }else{
		  console.log('not connected')
	  }
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
	      <ServerInput onSubmitFunction={this.updateServerAddress}/>
	    </Grid>
	    {this.mainContainer()}
	  </Grid>
        </div>
       </React.Fragment>
    );
  }
}

export default App;
