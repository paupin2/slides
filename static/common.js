const
	refFontSize = 10,
	warnFontSize = 40, // warn (make slide yellow) below this font size
	maxFontSize = 150,
	fontMultiplier = 0.9,
	slideWidth = 1280,
	slideHeight = 720,
	labelHeight = 110,
	refDiv = document.createElement('div');

refDiv.style = 'top:0;left:0;position:absolute;pointer-events:none;opacity:0;font-size:'+refFontSize+'pt';
document.body.appendChild(refDiv);

function log(msg) {
	if (console && console.log) {
		console.log(msg);
	}
}

function fitText(text, dims) {
	refDiv.innerText = text || '';
	const refWidth = refDiv.clientWidth,
		refHeight = refDiv.clientHeight;
	if (refWidth == 0 || refHeight == 0) {
		return refFontSize;
	}

	const fitWidth = (dims.width / refWidth) * refFontSize * fontMultiplier,
		fitHeight = (dims.height / refHeight) * refFontSize * fontMultiplier;

	return Math.min(fitWidth, fitHeight, maxFontSize);
}

function extend() {
	const extended = arguments.length ? (arguments[0] || {}) : {};
	for (let i = 1; i < arguments.length; i++) {
		const obj = arguments[i];
		for (let key in obj) {
			extended[key] = obj[key];
		}
	}

	return extended;
};

function Declick(delay) {
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
