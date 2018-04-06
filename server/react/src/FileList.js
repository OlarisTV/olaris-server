import React, { Component } from 'react'
import axios from 'axios';
import { withStyles } from 'material-ui/styles';
import Table, { TableBody, TableCell, TableHead, TableRow } from 'material-ui/Table';
import Paper from 'material-ui/Paper';

class FileList extends Component {
  state = { files: [] }

  componentDidMount() {
    axios.get("http://localhost:8080/api/v1/files").then(response => {
      this.setState({files: response.data})
    })
  }

  handleClick = (url, streamType) => {
    this.props.onClickMethod(url, streamType)
  };


  render() {
    return (
      <div className="filelist">
      Found {this.state.files.length} files.
      <Paper>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>FileSize</TableCell>
              <TableCell>Links</TableCell>
            </TableRow>
          </TableHead>
        <TableBody>
          {this.state.files.map(file =>
            <TableRow key={file.key}>
              <TableCell>{file.name}</TableCell>
              <TableCell>{file.size}</TableCell>
              <TableCell><a onClick={ ()=>{ this.handleClick(file.name) }} href="#">HLS Transcode</a></TableCell>
              <TableCell><a onClick={ ()=>{ this.handleClick(file.name, 'hls-transmuxing') }} href="#">HLS Transmux</a></TableCell>
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
