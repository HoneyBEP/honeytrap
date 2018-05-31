import React, { Component } from 'react';
import { connect } from 'react-redux';
import { bindActionCreators } from 'redux';
import { addSession, fetchSessions } from '../actions/index';
import { Link } from 'react-router';
import View from './view';

class ScriptOverview extends Component {
    constructor(props) {
        super(props);

        this.reloadScripts = this.reloadScripts.bind(this)
    }

    reloadScripts() {
        this.props.socket.websocket.send('{"action": "reload_scripts"}');
    }

    componentWillMount() {
        const { dispatch } = this.props;
    }

    renderTable() {

    }

    render() {
        if (!this.props.socket)
            return <div className="loading">Loading&#8230;</div>;

        return (
            <View title="Overview" subtitle="Scripts">
                <button className="btn btn-success" onClick={this.reloadScripts}>
                    Reload Scripts
                </button>
            </View>
        );
    }
}

function mapStateToProps(state) {
    return {
        socket: state.sessions.socket,
    };
}



export default connect(mapStateToProps)(ScriptOverview);