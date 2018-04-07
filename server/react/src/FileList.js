import React, { Component } from 'react'
import axios from 'axios';
import { withStyles } from 'material-ui/styles';
import Table, { TableBody, TableCell, TableHead, TableRow } from 'material-ui/Table';
import Paper from 'material-ui/Paper';
class Filter extends Component {
  state = { filterBy: '' }
}

class FileList extends Component {
  state = { files: [], name: '' }

  componentDidMount() {
    axios.get("http://localhost:8080/api/v1/files").then(response => {
      this.setState({files: response.data, allFiles: response.data})
    })
  }

  handleClick = (url, streamType) => {
    this.props.onClickMethod(url, streamType)
  };

  updateFilter = (event) => {
    if(event.target.value == "") {
      this.setState({name: event.target.value, files: this.state.allFiles})
    }else{
      let result = this.state.files.filter(file => ( file.name.toLowerCase().indexOf(event.target.value.toLowerCase()) > -1));
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
