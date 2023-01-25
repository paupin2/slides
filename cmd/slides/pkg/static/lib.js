function extend() {
	const extended = arguments.length ? (arguments[0] || {}) : {};
	for (let i = 1; i < arguments.length; i++) {
		const obj = arguments[i];
		for (let key in obj) {
			extended[key] = obj[key];
		}
	}

	return extended;
}

function log(msg) {
	if (console && console.log) {
		console.log(msg);
	}
}

function normalize(s) {
	return s.toLowerCase()
	.replace(/áâàãäå/, 'a')
	.replace(/ç/, 'c')
	.replace(/éê/, 'e')
	.replace(/íï/, 'i')
	.replace(/ôõöó/, 'o')
	.replace(/üú/, 'u');
}

function elem(tag, parent, className, text) {
	const el = (tag.indexOf('<') == -1) ? document.createElement(tag) : (function() {
		const e = document.createElement('div');
		e.innerHTML = tag;
		return e.firstChild;
	})();
	if (parent)	parent.appendChild(el);
	if (typeof className == 'string') {
		className.split(' ').forEach(c => {if (c) el.classList.add(c)});
	}
	if (typeof text == 'string') el.innerText = text;
	return el;
}

function showMessage(args) {
	const opt = extend({
		parent: document.body,
		timeout: 5000,
		kind: 'info',
		msg: ''
	}, args)
	const div = elem('<div class="alert">', opt.parent);
	div.innerText = args.msg;
	setTimeout(() => {
		div.classList.add('msg-'+opt.kind, 'show');
		setTimeout(() => {
			div.classList.remove('show');
			setTimeout(() => opt.parent.removeChild(div), 1000);
		}, opt.timeout);
	}, 25)
}

class Loading {
	static depth = 0;

	static start() {
		Loading.depth++;
		if (Loading.depth != 1) return;

		// start loading
		const bubbles = 20;
		const l = document.querySelector('#loading>div');
		if (l.childNodes.length == 0) {
			for (let i = 0; i < bubbles; i++) {
				l.appendChild(document.createElement('div'));
			}
		}

		// randomize box positions
		const w = document.body.clientWidth, h = document.body.clientHeight;
		const rnd = (min,max) => Math.floor(min+Math.random() * (max-min));
		const px = (n) => n + 'px';
		l.childNodes.forEach(n => {
			const radius = rnd(10,120);
			extend(n.style, {
				left: px(rnd(0,w)),
				backgroundColor: `#fff${rnd(1,16).toString(16)}`,
				animationDuration: rnd(5,20)+'s',
				bottom: px(-radius-rnd(0,200)),
				width: px(radius),
				height: px(radius),
			});
		});

		// start loading
		const bodyCL = document.body.classList;
		bodyCL.add('loading');
		setTimeout(() => bodyCL.add('load-animation'), 25);
	}

	static done() {
		Loading.depth--;
		if (Loading.depth > 0) return;
		const bodyCL = document.body.classList;
		bodyCL.remove('loading');
		setTimeout(() => bodyCL.remove('load-animation'), 25);
	}
}

function ajax(args) {
	const opt = extend({
		method: 'GET',
		path: '/',
		qs: '',
		data: null,
		success: null,
		failed: null
	}, args);
	var request = new XMLHttpRequest();
	if (opt.qs) {
		opt.path += "?" + (new URLSearchParams(opt.qs));
	}

	request.open(opt.method, opt.path, true);
	if (opt.data && (opt.method == 'POST' || opt.method == 'PUT')) {
		request.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');
	}

	function done(ok) {
		const end = performance.now();
		log(opt.method+' '+opt.path +': '+request.status+' '+request.statusText+(opt.data?' <data>':'')+': '+parseInt(end-started)+'ms');
		Loading.done();

		var reply = {}, parseOK = true;
		try { reply = JSON.parse(request.response); }
		catch (error) { parseOK = false; }

		const data = (reply && reply.data) ? reply.data : {};
		const statusOK = (request.status >= 200 && request.status < 400);
		if (ok && statusOK && parseOK && reply && reply.ok) {
			// success
			if (opt.success) {
				opt.success(data, request);
			}

		} else if (opt.failed) {
			// handle error manually
			opt.failed(data, request);

		} else if (reply && reply.error) {
			// show error
			showMessage({msg:reply.error, kind:'error'});
		}
	}

	Loading.start();
	request.onload = () => { done(true); };
	request.onerror = () => { done(false); };
	const started = performance.now();
	request.send(opt.data ? JSON.stringify(opt.data) : null);
}

// s.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})/)

function date(ref) {
	const _date = {
		year: 0,
		month: 0,
		day: 0,
		date() {
			return new Date(this.year, this.month-1, this.day);
		},
		toString() {
			return this.iso;
		},
		get fuzzy() {
			const diffms = (this.date()).getTime() - date().date().getTime();
			const diffdays = Math.abs(Math.round(diffms/1000/60/60/24));
			const diffweeks = Math.round(diffdays/7);
			const diffmonths = Math.floor(diffdays/30);
			const diffyears = Math.floor(diffdays/365);
			if (diffms >= 0) {
				if (diffdays == 0) return 'today';
				if (diffdays == 1) return 'tomorrow';
				if (diffdays <= 7) return 'next '+this.weekdayName;
				if (diffweeks == 1) return 'next week';
				if (diffyears == 1) return 'next year';
				if (diffyears > 0) return diffyears+' years from now';
				if (diffmonths > 8) return diffmonths+' months from now';
				return diffweeks + ' weeks from now';
			}
			if (diffdays == 1) return 'yesterday';
			if (diffdays <= 7) return 'last '+this.weekdayName;
			if (diffweeks == 1) return 'last week';
			if (diffyears == 1) return 'a year ago';
			if (diffyears > 0) return diffyears+' years ago';
			if (diffmonths > 8) return diffmonths+' months ago';
		return diffweeks + ' weeks ago';
		},
		get isLeapYear() {
			return !(this.year % 100) ? !(this.year % 400) : !(this.year % 4);
		},
		get monthName() {
			const names = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
			if (this.month < 1 || this.month > 12) return '';
			return names[this.month - 1];
		},
		get monthDays() {
			if (this.month == 2 && this.isLeapYear) return 29;
			const lengths = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
			return lengths[this.month - 1];
		},
		get weekday() {
			// Tomohiko Sakamoto Algorithm
			const t = [0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4];
			const y = this.year - (this.month < 3 ? 1 : 0);
			const m = this.month;
			const d = this.day;

			// 0.Sunday .. 6.Saturday
			const sunday0dow = (y + Math.floor(y / 4) - Math.floor(y / 100) + Math.floor(y / 400) + t[m - 1] + d) % 7;

			// convert to ISO 8601
			if (sunday0dow == 0) return 7;
			return sunday0dow;
		},
		get weekdayName() {
			const names = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
			return names[this.weekday - 1];
		},
		weekStart(day) {
			// ISO 8601: Monday = 1, Sunday = 7
			const wanted = (day >= 1 && day <= 7) ? Math.floor(day) : 1;
			const c = this.clone();
			while (c.weekday != wanted) c.moveDays(-1);
			return c;
		},
		get weekNumber() {
			// returns the ISO week of the date.
			const d = this.date();

			// Thursday in current week decides the year.
			d.setDate(d.getDate() + 3 - (d.getDay() + 6) % 7);

			// January 4 is always in week 1.
			const week1 = new Date(d.getFullYear(), 0, 4);

			// adjust to Thursday in week 1 and count number of weeks from date to week1.
			return 1 + Math.round(((d.getTime() - week1.getTime()) / 86400000 - 3 + (week1.getDay() + 6) % 7) / 7);
		},
		get iso() {
			const pad = (n) => ('00' + n).substr(-2);
			return this.year + '-' + pad(this.month) + '-' + pad(this.day);
		},
		get nextMonth() {
			return this.month == 12 ? 1 : this.month + 1;
		},
		get prevMonth() {
			return this.month == 1 ? 12 : this.month - 1;
		},
		get monthStart() {
			const c = this.clone();
			c.day = 1;
			return c;
		},
		set(y, m, d) {
			const iy = parseInt(y), im = parseInt(m), id = parseInt(d);
			if (iy < 1000 || iy > 3000) return false;
			if (im < 1 || im > 12) return false;
			this.year = iy;
			this.month = im;
			const max = this.monthDays;
			if (id < 1 || id > max) return false;
			this.day = id;
			return true;
		},
		setFromDate(dt) {
			this.set(dt.getFullYear(), dt.getMonth() + 1, dt.getDate());
		},
		setFromString(dt) {
			const m = dt.match(/^\d{4}-\d{2}-\d{2}/); // YYYY-MM-DD
			if (!m) {
				return false;
			}
			const ints = m[0].split('-').map(s => parseInt(s.replace(/^0/, '')));
			if (ints.length != 3 || !_date.set(ints[0], ints[1], ints[2])) {
				return false;
			}
			return true;
		},
		clone() {
			return new date(this);
		},
		moveDays(n) {
			let add = parseInt(n);
			if (add < 0) {
				let sub = -add;
				while (sub > 0) {
					if (sub < this.day) {
						this.day -= sub;
						sub = 0;

					} else {
						sub -= this.day;
						if (this.month == 1) this.year--;
						this.month = this.prevMonth;
						this.day = this.monthDays;
					}
				}

			} else {
				while (add > 0) {
					const daysLeftInMonth = this.monthDays - this.day;
					if (daysLeftInMonth > 0) {
						const days = Math.min(daysLeftInMonth, add);
						this.day += days;
						add -= days;

					} else {
						if (this.month == 12) this.year++;
						this.month = this.nextMonth;
						this.day = 1;
						add--;
					}
				}
			}
		},
		plusDays(n) {
			const c = this.clone();
			c.moveDays(n);
			return c;
		},
		moveMonths(n) {
			n = parseInt(n);
			while (n > 0) {
				n--;
				if (this.month < 12) {
					this.month++;
				} else {
					this.month = 1;
					this.year++;
				}
			}
			while (n < 0) {
				n++;
				if (this.month > 1) {
					this.month--;
				} else {
					this.month = 12;
					this.year--;
				}
			}
			const max = this.monthDays;
			if (this.day > max) this.day = max;
		},
		plusMonths(n) {
			const c = this.clone();
			c.moveMonths(n);
			return c;
		}
	};

	if (ref && ref.year && ref.month && ref.day) {
		_date.set(ref.year, ref.month, ref.day);
	} else if (ref && typeof ref.getMonth === 'function') {
		_date.setFromDate(ref);
	} else if (ref && typeof ref === 'string') {
		if (!_date.setFromString(ref)) {
			return null;
		}

	} else {
		_date.setFromDate(new Date());
	}

	return _date;
}

const refFontSize = 10;
const refDiv = elem('<span style="display:inline-block;opacity:0;top:0;left:0;position:absolute;pointer-events:none;white-space:pre-line;font-size:'+refFontSize+'pt">', document.body);

function fitText(args) {
	const opt = extend({
		text: '',
		width: 0,
		height: 0,
		mult: 1,
		min: 10,
		max: 150
	}, args || {});
	// set the reference div text, let the browser calculate its size
	refDiv.innerText = opt.text;
	const refWidth = refDiv.clientWidth;
	const refHeight = refDiv.clientHeight;
	if (refWidth == 0 || refHeight == 0) {
		return refFontSize;
	}

	const maxFontSize = opt.height/4;
	const fitWidth = (opt.width / refWidth) * refFontSize * opt.mult;
	const fitHeight = (opt.height / refHeight) * refFontSize * opt.mult;
	const fontSize = Math.max(opt.min, Math.min(fitWidth, fitHeight, maxFontSize));
	// log('fitting '+opt.text.length+' chars to '+opt.width+'x'+opt.height+': '+fontSize+'pt')
	return fontSize;
}
