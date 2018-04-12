import React, { Component } from 'react'
import axios from 'axios';
import Table, { TableBody, TableCell, TableHead, TableRow } from 'material-ui/Table';
import Paper from 'material-ui/Paper';

class Filter extends Component {
  state = { filterBy: '' }
}

class FileList extends Component {
  state = { files: [], name: '' }

  componentDidMount() {
    axios.get(`${this.props.serverAddress}/api/v1/files`).then(response => {
      this.setState({files: response.data, allFiles: response.data})
    })
  }

  handleClick = (url, name, playtime) => {
    this.props.onClickMethod(url, name, playtime)
  };

  updateFilter = (event) => {
    if(event.target.value === "") {
      this.setState({name: event.target.value, files: this.state.allFiles})
    }else{
      let result = this.state.allFiles.filter(file => ( file.name.toLowerCase().indexOf(event.target.value.toLowerCase()) > -1));
      this.setState({name: event.target.value, files: result})
    }
  }


  render() {
    return (
      <div className="filelist">
      Found {this.state.files.length} files.
      <input value={this.state.name} onChange={this.updateFilter} placeholder="Filter" />
      <Paper>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Playtime</TableCell>
              <TableCell>Links</TableCell>
            </TableRow>
          </TableHead>
        <TableBody>
          {this.state.files.map(file =>
            <TableRow key={file.key}>
              <TableCell>{file.name}</TableCell>
              <TableCell>{file.playtime}</TableCell>
              <TableCell><a onClick={ (e)=>{ e.preventDefault(); this.handleClick(this.props.serverAddress+ file.hlsTranscodingManifest, file.name, file.playtime) }} href="#">HLS Transcode</a></TableCell>
              <TableCell><a onClick={ (e)=>{ e.preventDefault(); this.handleClick(this.props.serverAddress+ file.hlsTransmuxingManifest, file.name, file.playtime) }} href="#">HLS Transmux</a></TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      </Paper>
      </div>
    )
  }
}

export default FileList;
