var channelStore = require('./stores/channel');
var channelActions = require('./actions/channel');
var messageActions = require('./actions/message');
var serverActions = require('./actions/server');
var tabActions = require('./actions/tab');

messageActions.command.listen(function(line, channel, server) {
	var params = line.slice(1).split(' ');
	var command = params[0].toLowerCase();

	switch (command) {
		case 'nick':
			if (params[1]) {
				serverActions.setNick(params[1], server);
			}
			break;

		case 'quit':
			serverActions.disconnect(server);
			break;

		case 'join':
			if (params[1]) {
				channelActions.join([params[1]], server);
				tabActions.select(server, params[1]);
			}
			break;

		case 'part':
			if (channel) {
				channelActions.part([channel], server);
			}
			break;

		case 'me':
			if (params.length > 1) {
				messageActions.send('\x01ACTION ' + params.slice(1).join(' ') + '\x01', channel, server);
			}
			break;

		case 'topic':
			var topic = channelStore.getTopic(server, channel);
			if (topic) {
				messageActions.inform(topic, server, channel);
			} else {
				messageActions.inform('No topic set', server, channel);
			}
			break;
	}
});