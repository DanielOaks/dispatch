var Reflux = require('reflux');
var _ = require('lodash');

var serverStore = require('./server');
var channelStore = require('./channel');
var actions = require('../actions/message');
var serverActions = require('../actions/server');
var channelActions = require('../actions/channel');

var messages = {};

function addMessage(message, dest) {
	message.time = new Date();

	if (message.message.indexOf('\x01ACTION') === 0) {
		var from = message.from;
		message.from = null;
		message.type = 'action';
		message.message = from + message.message.slice(7);
	}

	if (!(message.server in messages)) {
		messages[message.server] = {};
		messages[message.server][dest] = [message];
	} else if (!(dest in messages[message.server])) {
		messages[message.server][dest] = [message];
	} else {
		messages[message.server][dest].push(message);
	}
}

var messageStore = Reflux.createStore({
	init: function() {
		this.listenToMany(actions);
		this.listenTo(serverActions.disconnect, 'disconnect');
		this.listenTo(channelActions.part, 'part');
	},

	send: function(message, to, server) {
		addMessage({
			server: server,
			from: serverStore.getNick(server),
			to: to,
			message: message
		}, to);

		this.trigger(messages);
	},

	add: function(message) {
		var dest = message.to || message.from;
		if (message.from && message.from.indexOf('.') !== -1) {
			dest = message.server;
		}

		addMessage(message, dest);
		this.trigger(messages);
	},

	broadcast: function(message, server, user) {
		_.each(channelStore.getChannels(server), function(channel, channelName) {
			if (!user || (user && _.find(channel.users, { nick: user }))) {
				addMessage({
					server: server,
					to: channelName,
					message: message,
					type: 'info'
				}, channelName);
			}
		});
		this.trigger(messages);
	},

	inform: function(message, server, channel) {
		addMessage({
			server: server,
			to: channel,
			message: message,
			type: 'info'
		}, channel || server);
		this.trigger(messages);
	},

	disconnect: function(server) {
		delete messages[server];
		this.trigger(messages);
	},

	part: function(channels, server) {
		_.each(channels, function(channel) {
			delete messages[server][channel];
		});
		this.trigger(messages);
	},

	getMessages: function(server, dest) {
		if (messages[server] && messages[server][dest]) {
			return messages[server][dest];
		}
		return [];
	},

	getState: function() {
		return messages;
	}
});

module.exports = messageStore;