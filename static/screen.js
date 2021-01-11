const refreshInterval = 500,
	retryInterval = 2 * 1000,
	body = document.body,
	container = document.querySelector('#container'),
	ref = document.querySelector('#ref');

let refreshTimeout = Declick(refreshInterval);
let resizeTimeout = Declick(250);
let checksum = Math.floor(Math.random() * 4294967296);
const dims = () => ({width:window.innerWidth, height:window.innerHeight});

window.onresize = function() {
	resizeTimeout(function() {
		const size = fitText(container.innerText, dims());
		container.style.fontSize = size + 'pt';
	});
}

function update(text) {
	const size = fitText(text, dims());
	container.style.fontSize = size + 'pt';
	container.innerText = text;
	log('updated to "' + text.replace(/\n/g, ' ').substr(0, 20) + '…" (cksum ' + checksum + ') at ' + size + 'pt');
}

function getUpdate() {
	$.ajax('refresh.json', {
		data: {cksum: checksum},
		method: 'GET',
		dataType: 'json',
		success(data) {
			const text = data.show || '';
			const cksum = data.cksum || 0;
			if (cksum == checksum) {
				return;
			}
			checksum = cksum;
			update(text);
		},
		complete() {
			refreshTimeout(getUpdate);
		},
	});
}

let retryTimeout = Declick(retryInterval);
function initSocket() {
	const loc = document.location;
	const protocol = loc.protocol == 'https:' ? 'wss:' : 'ws:';
	const address = protocol + '//' + loc.host + loc.pathname + '.socket';
	const conn = new WebSocket(address);
	extend(conn, {
		onopen() {
			log('opened socket to ' + address);
		},
		onclose() {
			log('socket closed');
			retryTimeout(initSocket);
		},
		onerror() {
			log('socket error');
			retryTimeout(initSocket);
		},
		onmessage(evt) {
			let slide;
			try {
				const data = JSON.parse(evt.data);
				slide = data.show || '';
			} catch (error) {
				return;
			}
			update(slide);
		},
	});
}

update('connecting');

function toggleFullScreen(e) {
	if (body.requestFullscreen) {
		body.requestFullscreen();
	} else if (body.webkitRequestFullscreen) {
		body.webkitRequestFullscreen();
	} else if (body.msRequestFullscreen) {
		body.msRequestFullscreen();
	}
	e.preventDefault();
}

body.addEventListener('dblclick', toggleFullScreen);
container.addEventListener('dblclick', toggleFullScreen);

// start updating, either through websockets (if available) or ajax requests
if (window.WebSocket) {
	initSocket();
} else {
	getUpdate();
}
