const
	// configuration
	previewScale = .2, // scale of slides in the preview list
	updateDelay = 250, // ms before updating slides
	px = (v) => v + 'px',
	pt = (v) => v + 'pt',

	// elements
	editor = query("#editor"),
	body = document.body,

	showHelp = newToggle((on) => body.classList.toggle('nohelp', !on), true),

	// regexes
	reChord = /^[A-G](##?|bb?)?((m|sus|maj|min|aug|dim)?\d?)?\.?$/,
	reTitle = /^(?:([a-zåäö0-9]+(?:\s+[a-zåäö0-9]+)?):$|^#+(.*)|^\[?((?:intro|chorus|bridge|verse)(?:\s*\d+)?)\]?$)/i,
	reCleanups = [
		[/\(repeat.*?\)/gi, ''],
		[/\bcolumn_break\b/gi, ''],
		[/\[[A-G](##?|bb?)?((m|sus|maj|min|aug|dim)?\d?)?\.?\]/gi, '']
	];

let webSocket = null;

const Slide = (text, label, start, end) => ({
	text: text,
	fontSize: fitText(text, {
		width: slideWidth,
		height: label ? labelHeight : slideHeight,
	}),
	label: !!label,
	element: null,
	offset: {start:start, end:end},
	highlight() {
		// scroll into view
		// if not a label, also highlight
		const scrollExtra = 20;
		const highlightFade = 500;
		if (!this.element) {
			return;
		}

		const parent = this.element.parentNode,
			visibleTop = parent.scrollTop,
			visibleBottom = visibleTop + parent.clientHeight,
			elementTop = this.element.offsetTop,
			elementBottom = elementTop + this.element.clientHeight;

		if (elementBottom > visibleBottom) {
			// below visible area
			parent.scrollTop += (elementBottom - visibleBottom + scrollExtra);

		} else if (elementTop < visibleTop) {
			// above visible area
			parent.scrollTop -= (visibleTop - elementTop + scrollExtra);
		}

		const current = deck.ul.querySelector('.focus');
		if (current != this.element) {
			if (current) current.classList.remove('focus');
			this.element.classList.add('focus');
		}
	},
});

const deck = {
	ul: query('#preview'),
	slides: [],
	selected: -1,
	lastHighlighted: null,
	findElement(elm) {
		for (let i = 0; i < this.slides.length; i++) {
			if (this.slides[i].element == elm) {
				return i;
			}
		}
		return -1;
	},
	replace(newSlides) {
		// save the selection
		const selIndexBefore = this.selected,
		selTextBefore = (this.selected == -1) ? '' : this.slides[this.selected].text;

		// replace all items
		this.slides.splice(0, this.slides.length);
		this.ul.innerHTML = '';
		newSlides.forEach(s => this.addSlide(s));

		if (selTextBefore == '') {
			return;
		}

		// try to select the slide that was previously selected before
		const matches = [];
		for (let i = 0; i < this.slides.length; i++) {
			if (this.slides[i].text == selTextBefore) {
				matches.push({
					index: i,
					distance: Math.abs(i - selIndexBefore)
				});
			}
		}
		matches.sort((a, b) => a.distance > b.distance);
		if (matches.length > 0) {
			this.select(matches[0].index)
			return;
		}

		// not possible; unselect
		this.select(-1);
	},
	cleanup() {
		let text = '';
		textToSlides(editor.value).forEach(function(t) {
			if (t.label) {
				text += "# " + t.text + "\n";
			} else {
				text += t.text + "\n\n";
			}
		});
		editor.value = text;
	},
	select(index) {
		const selBefore = this.ul.querySelector('.slide.selected');
		if (selBefore) selBefore.classList.remove('selected');

		this.selected = index;
		if (this.selected != -1) {
			this.slides[this.selected].element.classList.add('selected');
		}
		if (this.selected == -1) {
			show('');

		} else {
			const item = this.slides[this.selected];
			show(item.text);
		}
	},
	addSlide(slide) {
		const elm = document.createElement('li');
		slide.element = elm;

		const height = slide.label ? labelHeight : slideHeight;
		elm.classList.toggle('slide', !slide.label);
		elm.classList.toggle('label', slide.label);
		extend(elm.style, {
			width: px(slideWidth * previewScale),
			height: px(height * previewScale),
			fontSize: pt(slide.fontSize * previewScale),
		});
		if (!slide.label) {
			onclick(elm, () => this.select(this.findElement(elm)));
		}

		slide.text.split("\n").forEach(function(line, idx) {
			if (idx > 0) elm.appendChild(document.createElement('br'));
			elm.appendChild(document.createTextNode(line));
		})

		this.ul.appendChild(elm);
		this.slides.push(slide);
		return slide;
	},
	findPos(pos) {
		// use a binary search to find in which slide `pos` falls
		for (let first = 0, last = this.slides.length - 1; first <= last; ) {
			const middle = Math.floor((first+last)/2);
			const slide = this.slides[middle];
			if (pos < slide.offset.start) {
				// try again in first half
				last = middle - 1;

			} else if (pos > slide.offset.end) {
				// try again in second half
				first = middle + 1;

			} else {
				return slide;
			}
		}
		return null;
	},
	highlightPos(pos, force) {
		const slide = this.findPos(pos);
		if (slide && (this.lastHighlighted != slide.element || force)) {
			this.lastHighlighted = slide.element;
			slide.highlight();
		}
	}
}

function query(s) {
	return document.querySelector(s);
}

function newToggle(cb, initial) {
	let value = initial || false;
	return function(force) {
		const before = value;
		if (typeof(force) == 'boolean') value = force;
		else value = !value;
		if (value != before) cb(value);
	}
}

function onclick(selector, cb) {
	const elm = (typeof(selector) == 'string' ? query(selector) : selector);
	elm.addEventListener('click', function(evt) {
		evt.preventDefault();
		const down = !this.classList.contains('down');
		if (this.classList.contains('toggle')) {
			this.classList.toggle('down', down);
		}
		if (cb) {
			cb.apply(this, [down, this]);
		}
	});
}

function save(text) {
	const data = {text:text};
	log('PUTing ' + JSON.stringify(data));
	$.ajax('save.json', {
		data: JSON.stringify(data),
		method: 'PUT',
		dataType: 'json',
		success(data) {

		},
		complete() {

		},
	});
}

function show(text) {
	const data = {show: text || ''};
	log('PUTing ' + JSON.stringify(data));
	$.ajax('show.json', {
		data: JSON.stringify(data),
		method: 'PUT',
		dataType: 'json'
	});
}

function isChordLine(line) {
	const words = line.split(/[\s/]+/);
	let anyChord = false;
	for (let i = 0; i < words.length; i++) {
		const word = words[i];
		if (word) {
			if (!word.match(reChord)) {
				return false;
			}
			anyChord = true;
		}
	}
	return anyChord;
}

function textToSlides(text) {
	// split text into slides
	// first, each line is trimmed for spaces/tabs
	// then, each block of text with at least one empty line between them
	// is considered a separate slide, similar to markdown

	const slides = [];
	const current = [];
	let currentStartOffset = 0;
	let startOffset = 0, endOffset = 0;

	const addCurrent = function() {
		if (current.length) {
			const text = current.join("\n");
			slides.push(Slide(text, false, currentStartOffset, endOffset));
			current.splice(0, current.length);
			currentStartOffset = endOffset+1;
		}
	}

	const parse = function() {
		// clean line
		let line = text.substring(startOffset-1, endOffset).trim();
		for (let i = 0; i < reCleanups.length; i++) {
			const re = reCleanups[i][0], repl = reCleanups[i][1];
			line = line.replace(re, repl);
		}

		// chord line? ignore
		if (isChordLine(line)) {
			return;
		}

		// label/empty line? start a new slide
		const match = line.match(reTitle);
		if (match || !line.length) {
			addCurrent();
			(match || []).forEach((s, i) => {
				if (i > 0 && s !== undefined) {
					// use the first defined group, after the first,
					// as the label text
					slides.push(Slide(s.trim(), true, startOffset, startOffset));
				}
			});
			return;
		}

		// normal line? push to current slide
		current.push(line);
	}

	for (const chr of text) {
		endOffset++;
		if (chr == "\n") {
			parse();
			startOffset = endOffset + 1;
		}
	}
	parse();
	addCurrent();
	slides.push(Slide('end', true, startOffset, endOffset));
	return slides;
}

function init() {
	deck.replace(textToSlides(editor.value));
}

let updateLastValue = '';
let updateLastTexts = [];
let updateTimeoutID = null;

function update(now, dontSend) {
	showHelp(editor.value == '');

	if (now !== true) {
		// don't update now, set a timeout instead, for de-clicking
		if (updateTimeoutID) {
			clearTimeout(updateTimeoutID);
		}
		updateTimeoutID = setTimeout('update(true)', updateDelay);
		return;
	}
	updateTimeoutID = null;

	// is the string value equal to last time?
	if (editor.value == updateLastValue) return;
	updateLastValue = editor.value;

	// split into lines; is the array equal to last time?
	const texts = textToSlides(editor.value);
	if (texts.length == updateLastTexts.length) {
		let different = false;
		for (let i = 0; i < texts.length; i++) {
			if (texts[i].text != updateLastTexts[i]) {
				// different text
				different = true;
				break;
			}
		}
		if (!different) return;
	}
	updateLastTexts = texts;
	if (!dontSend) {
		save(editor.value);
	}

	// different text
	deck.replace(texts);
}

// on keyup/change, update the slides
editor.addEventListener('keyup', update);
editor.addEventListener('change', update);
editor.addEventListener('keyup', (e) => deck.highlightPos(e.target.selectionEnd));
editor.addEventListener('mouseup', (e) => deck.highlightPos(e.target.selectionEnd, true));

onclick('#clear', () => deck.select(-1));
onclick('#show-ed', (on) => editor.disabled = !on);
onclick('#show-slides', function(on) {
	body.classList.toggle('noslides', !on);
	if (!on && body.classList.contains('noeditor')) {
		body.classList.remove('noeditor');
		query('#show-ed').classList.add('down');
	}
});
onclick('#cleanup', () => deck.cleanup());
onclick('#show-help', (on) => showHelp(on));

// initial slides
jQuery.getJSON('load.json', function(data) {
	if (data) {
		const text = data.text || '';
		log('loaded "' + text.replace(/\n/g, ' ').substring(0, 20) + '…"');
		editor.value = text;
		update(true, true);
	}
});
