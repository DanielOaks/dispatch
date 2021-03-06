import React, { Component } from 'react';
import pure from 'pure-render-decorator';

@pure
export default class TabListItem extends Component {
  handleClick = () => {
    const { server, target, onClick } = this.props;
    onClick(server, target);
  };

  render() {
    const { target, content, selected } = this.props;
    const classes = [];

    if (!target) {
      classes.push('tab-server');
    }

    if (selected) {
      classes.push('selected');
    }

    let indicator = null;
    if (this.props.connected !== undefined) {
      const style = {};

      if (this.props.connected) {
        style.background = '#6BB758';
      } else {
        style.background = '#F6546A';
      }

      indicator = <i className="tab-indicator" style={style}></i>;
    }

    return (
      <p className={classes.join(' ')} onClick={this.handleClick}>
        <span className="tab-content">{content}</span>
        {indicator}
      </p>
    );
  }
}
