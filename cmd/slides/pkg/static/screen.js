function declick(delay) {
	let timeout = null;

	return function(callback, newDelay) {
		if (timeout != null) {
			clearTimeout(timeout);
		}
		timeout = setTimeout(function() {
			timeout = null;
			if (callback && callback.call) {
				callback.call();
			}
		}, newDelay || delay || 500);
	}
}

const body = document.body,
	container = document.querySelector('#container'),
	ref = document.querySelector('#ref');

let lastText = '';

const nbsp = (function() {
	const elm = document.createElement('div');
	elm.innerHTML = '&nbsp;';
	return elm.textContent;
})();

function updateText() {
    const padding = 20;
    const dims = {
		width: window.innerWidth-padding,
		height: window.innerHeight-padding
	};

	let text = lastText;
	const subtitles = text.split('\n')[0] == '_';
	if (subtitles) {
		const padding = '\n' + nbsp + '\n' + nbsp;
		text = (text+padding).split('\n').slice(1, 3).join('\n');
		dims.height = dims.width / 8;
	}

	const size = fitText(extend(dims, {text:text}));
	container.style.fontSize = size + 'pt';
	container.innerText = text;
	document.body.classList.toggle('subtitles', subtitles);
}

function update(text) {
	lastText = text || '';
	updateText();
	log('updated to "' + text.replace(/\n/g, ' ').substr(0, 20) + '…"');
}

function bodyclass(cl, on) {
    body.classList.toggle(cl, on);
}

let messages = 0;
function ping() {
    messages++;
    if (messages == 1) {
        bodyclass('message', true);
    }
    setTimeout(() => {
        messages--;
        if (messages == 0) {
            bodyclass('message', false);
        }
    }, 1000);
}

let retryTimeout = declick(2 * 1000);
function initSocket() {
	const loc = document.location;
	const protocol = loc.protocol == 'https:' ? 'wss:' : 'ws:';
	const address = protocol + '//' + loc.host + '/screen' + loc.search;
	const conn = new WebSocket(address);
	extend(conn, {
		onopen() {
			log('opened socket to ' + address);
            bodyclass('connected', true);
		},
		onclose() {
			log('socket closed');
            bodyclass('connected', false);
			retryTimeout(initSocket);
		},
		onerror() {
            log('socket error');
            bodyclass('connected', false);
			retryTimeout(initSocket);
		},
		onmessage(evt) {
            ping();
			let slide = '';
			try {
				const data = JSON.parse(evt.data);
				slide = data.text || '';
			} catch (error) {
				return;
			}
			update(slide);
		},
	});
}

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

let resizeTimeout = declick(250);
window.onresize = () => { resizeTimeout(updateText); }
body.addEventListener('dblclick', toggleFullScreen);
container.addEventListener('dblclick', toggleFullScreen);

// start the websocket
update('connecting');
initSocket();