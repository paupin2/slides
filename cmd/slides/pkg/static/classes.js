const SongExtraFields = {
    author: 'Author',
    ccli: 'CCLI',
}

class Recent {
    static titles = [];
    static rows =  [];

    static update() {
        if (!(Deck.All.length && Song.All.length)) return;

        const used_songs = {};
        const recent_limit = 25;
        const recent_decks = [];

        // add the most recent items with songs
        for (let i = 0, l = Deck.All.length; i < l; i++) {
            const deck = Deck.All[i];
            if (!deck.songs.length) continue;
            deck.songs.forEach(song_id => used_songs[song_id] = true);
            recent_decks.push(deck);
            if (recent_decks.length >= recent_limit) break;
        }

        // build the table
        const titles = recent_decks.map(d => d.fuzzy ? d.fuzzy : d.title);
        Recent.titles.splice(0, Recent.titles.length, ...titles);
        Recent.rows.splice(0, Recent.rows.length);

        for (let song_id in used_songs) {
            if (!Song.ById[song_id]) continue;
            Recent.rows.push({
                song_id: song_id,
                song: Song.ById[song_id],
                cells: recent_decks.map(deck => deck.songs.indexOf(parseInt(song_id)) != -1)
            });
        }
    }
}

class Song {
    static All = [];
    static ById = {};
    static Recent = {};

    static refresh(callback) {
        ajax({path:"/songs", success: (data) => {
            const items = (data || []).map(s => new Song(s));
            Song.All.splice(0, Song.All.length, ...items);
            Song.ById = {};
            items.forEach(s => Song.ById[s.id] = s);

            if (callback) {
                callback.apply(this, [this]);
            }
            Recent.update();
        }});
    }

    /** create a new draft */
    static draft() {
        const song = new Song();
        Song.All.push(song);
        return song;
    }

    constructor(data) {
        data = data || {};
        this.id = data.id || null;
        for (let field in SongExtraFields) {
            this[field] = null;
        }
        this.imported = false;
        this.modified = null;
        this.slides = [];
		this.loaded = false;
        this.title = '';
        this.text = '';
        this.initialText = '';
        this.slidesText = '';
        this.change(extend({
            title: 'Song title',
            text: 'Song text',
        }, data));
	}

    /** updateFromPrefix updates fields from the initial headers, and returns
     * the position of first non-field line.
     */
    updateFromPrefix() {
        let index = 0, foundTitle = false;
        while (true) {
            const start = this.text.indexOf('#', index);
            if (start != index) break;

            const end = this.text.indexOf('\n', index+1);
            if (end == -1) break;
            index = end+1; // point to the start of the next line

            const line = this.text.substring(start+1, end).trim();
            const parts = line.match(/^\s*([a-z0-9]+)\s*:\s*(.*)/i);
            if (parts) {
                const field = parts[1].toLowerCase(), value = parts[2];
                if (SongExtraFields[field]) {
                    this[field] = value;
                    continue;
                }
            }

            if (foundTitle) break; // found a non-field line after the title
            this.title = line;
            foundTitle = true;
        }
        return index;
    }

    /** update title and slides from text */
    update() {
        const text = this.text.substring(this.updateFromPrefix());

        // update slides
        if (this.slidesText != text) {
            this.slides = Slide.Parse(text, true);
            this.slidesText = text;
        }
    }

    /** textOnly returns the song's text without the title (as the first header) */
    get textOnly() {
        if (this.text.match(/^\s*#/)) {
            // has a header as the first line: strip it
            return this.text.split('\n').slice(1).join('\n');
        }
        return this.text;
    }

    /** pasteText returns the text, prefixed by a title with the id */
    get pasteText() {
        const title = this.title + (this.id ? ` (@${this.id})` : '');
        const text = this.text.substring(this.updateFromPrefix());
        return `# ${title}\n${text}\n`;
    }

	change(data) {
        if (data.title) this.title = data.title;
        if (data.author) this.author = data.author || null;
        if (data.ccli) this.ccli = data.ccli || null;
        if (data.imported) this.imported = !!data.imported;
        if (data.text) {
            let extra = `# ${this.title}\n`;
            for (let field in SongExtraFields) {
                extra += `# ${SongExtraFields[field]}: ${this[field] || ''}\n`;
            }
            this.text = `${extra}${data.text}\n`;
            this.initialText = this.text;
        }
        if (data.modified) this.modified = dayjs(data.modified);
    }

    load(callback) {
		if (this.loaded || !this.id) {
			if (callback) callback(this);
			return;
		}

		ajax({path:"/song", qs:{song_id:this.id}, success: (data) => {
			this.loaded = true;
            this.dirty = false;

			this.change(data);
            this.update();
            if (callback) callback(this);
		}});
	}
	save(callback) {
        if (!this.dirty) return;

        // remove title and other fields from text before saving
        const start = this.updateFromPrefix();
        let saveText = this.text.substring(start);

        const inserting = !this.id;
		const method = inserting ? 'POST' : 'PUT';
		const data = {
			title:this.title,
			text:saveText,
		};
        for (let field in SongExtraFields) {
            data[field] = this[field];
        }

        if (this.id) data.id = this.id;

		ajax({method:method, data:data, path:'/song', success:(song)=> {
			showMessage({msg:`saved "${this.title}"`});
			this.change(song);
            this.dirty = false;

			// add item to the list, if it's not there
            if (inserting) {
                Song.ById[this.id] = this;
            }

			if (callback) callback(true, this);
		}, failed: ()=> {
			showMessage({kind:'error', msg:`error saving "${this.title}"`});
			if (callback) callback(false, this);
		}});
	}

    revert() {
        if (!this.dirty) return;
        this.text = this.initialText;
        this.dirty = false;
        this.update();
    }
}

class Tab {
	constructor(args, parent) {
		const opt = extend({
			kind: 'generic',
			icon: '',
			title: '',
			onfocus: null,
			onblur: null,
			props: {},
			app: {}
		}, args);

		if (!opt.app.props) opt.app.props = [];
		opt.app.props.push('tab', 'tabs');

		const titleClasses = [];
		if (opt.icon) {
			titleClasses.push('has-icon', 'i-'+opt.icon);
		}

		this.parent = parent;
		this.kind = opt.kind;
		this.handlers = {focus:opt.onfocus, blur:opt.onblur};
		this.vue = null;

		this.tabTitleElement = elem('li', null, titleClasses.join(' '), opt.title);
		this.tabPaneElement = elem('div', null, `tab-pane ${opt.kind}-pane `);
        this.parent.attach(this);

        this.app = Vue.createApp(opt.app, extend(opt.props, {tab:this, tabs:this.parent}));
		setTimeout(() => { this.vue = this.app.mount(this.tabPaneElement); }, 10);

		// set content, add event handlers
		this.tabTitleElement.addEventListener('mousedown', (e) => {
			e.preventDefault();
			this.parent.active = this;
		});

		// make active if first
		if (!this.parent.activeItem) {
			this.parent.active = this;
		}
	}
	get active() {
		return this.parent.active == this;
	}
	get title() {
		return this.tabTitleElement.innerText;
	}
	set title(text) {
		this.tabTitleElement.innerText = text;
	}
	set active(on) {
		this.tabTitleElement.classList.toggle('active', on);
		this.tabPaneElement.classList.toggle('active', on);
		this.handle(on ? 'focus' : 'blur');
	}
	show() {
		this.parent.active = this;
	}
	index() {
		let index = -1;
		this.items.forEach((t, i) => {
			if (t == this) {
				index = i;
			}
		});
		return index;
	}
	next() {
		let idx = this.index();
		if (idx != -1 && idx < this.items.length-1) {
			return this.items[idx+1];
		}
		return null;
	}
	prev() {
		let idx = this.index();
		if (idx > 0) {
			return this.items[idx-1];
		}
		return null;
	}
	close() {
		// unmount app, remove elements
        this.parent.detach(this);
		this.app.unmount();
	}
	handle(eventName) {
		if (this.handlers[eventName]) {
			this.handlers[eventName].apply(this, [this, eventName]);
		}
	}
}

class Tabs {
    constructor() {
        this.items = [];
        this.tabTitlesElement = document.querySelector('.tab-line');
        this.tabPanesElement = elem('div', document.querySelector('#container'), 'pane-container');
        this.activeItem = null;
        this.previous = null;
    }
    add(args) {
        return new Tab(args, this);
    }
    attach(tab) {
        this.items.push(tab);
        this.tabTitlesElement.appendChild(tab.tabTitleElement);
        this.tabPanesElement.appendChild(tab.tabPaneElement);
    }
    detach(tab) {
        // unselect the tab if it's the active one
        if (this.active == tab) {
            this.active = null;
        }

        // remove it from the items
        this.items.splice(tab.index(), 1);
        this.tabTitlesElement.removeChild(tab.tabTitleElement);
		this.tabPanesElement.removeChild(tab.tabPaneElement);
    }
    find(kind, title) {
        return this.items.find(i => i.kind == kind && i.title == title);
    }
    close(tab) {
        return this.items.find(i => i.kind == kind && i.title == title);
    }
    get active() {
        return this.activeItem;
    }
    set active(tab) {
        if (this.activeItem) {
            this.activeItem.active = false;
            if (tab == null) {
                // when unselecting a tab, select the previous one
                if (this.previous && this.previous != this.activeItem) tab = this.previous;
                else tab = this.activeItem.next() || this.activeItem.prev();
            }
        }

        this.previous = this.activeItem;
        this.activeItem = tab;
        if (tab) {
            tab.active = true;
        }
    }
}

const ThumbnailHeight = 135;
const ThumbnailWidth = ThumbnailHeight*16/9;
const LabelHeight = ThumbnailHeight/4;

class Slide {
    constructor(text, start, end, headers) {
        this.headers = headers ? [...headers] : [];
        this.text = text;
        this.thumbnailText = '';
        this.start = start;
        this.classes = ['slide'];
        this.style = {};
        this.end = end;
    }

    get isSubtitle() {
        return this.text.split('\n')[0] == '_';
    }

    /** measure ensures the slide can be used as a thumbnail */
    measure() {
        if (this.thumbnailText) return;

        const dims = {
            width:ThumbnailWidth,
            height:ThumbnailHeight,
        };
        this.thumbnailText = this.text;
        const sub = this.isSubtitle;
        if (sub) {
            // hasSubtitles: get only first 2 lines, pad with empty lines if needed
            this.thumbnailText = (this.text+"\n\xa0\n\xa0").split('\n').slice(1, 3).join('\n');
            dims.height = dims.width / 8;
            this.classes.push('subtitles');
        }

        extend(dims, {
            min:4,
            text:this.thumbnailText,
        })
        extend(this.style, {
            'font-size': fitText()+'pt',
        });
    }

    /** Cleanup parses the text again, removing anything that doesn't appear on slides */
    static Cleanup(source) {
        let text = '';
        Slide.Parse(source).forEach((slide) => {
            slide.headers.forEach(h => text += `# ${h.trim()}\n`);
            text += slide.text.trim() + "\n\n";
        });
        return text;
    }

    /** Parse converts the text to slides */
    static Parse(text, measure) {
        // split text into slides
        // first, each line is trimmed for spaces/tabs
        // then, each block of text with at least one empty line between them
        // is considered a separate slide, similar to markdown

        function isChordLine(line) {
            const reChord = /^[A-G](##?|bb?)?((m|sus|maj|min|aug|dim)?\d?)?\.?$/;
            const words = line.split(/[\s/|]+/);
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

        function addCurrent() {
            if (currentLines.length) {
                const text = currentLines.join("\n");
                slides.push(new Slide(text, currentStartOffset, endOffset, currentHeaders));
                currentHeaders.splice(0, currentHeaders.length);
                currentLines.splice(0, currentLines.length);
                currentStartOffset = endOffset;
            }
        }

        function parseLine() {
            const reCleanups = [
                [/\(repeat.*?\)/gi, ''],
                [/\bcolumn_break\b/gi, ''],
                [/\[[A-G](##?|bb?)?((m|sus|maj|min|aug|dim)?\d?)?\.?\]/gi, ''],
                // [G ///  | C2/G/ |]
                // [G ///  | C2/G/ |]
                // [G///   | C2///   | 2x|]
                // You [Dadd4]face
                [/\[(([A-G][a-z0-9]{0,4}|[0-9]x)[|/\s]*)+\]/, ''],
                [/\s{2,}/g, ' '],
                [/ +- +/g, ''] // join syllable split: "sna - ror" -> "snaror"
            ];
            const reTitle = /^(?:([a-zåäö0-9]+(?:\s+[a-zåäö0-9]+)?):$|^#+(.*)|^\[?((?:intro|outro|chorus|bridge|verse)(?:\s*\d+)?(?:\s*[0-9]x)?)\]?$)/i;

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

            // empty line? start a new slide
            const match = line.match(reTitle);
            if (match || !line.length) {
                addCurrent();
                (match || []).forEach((s, i) => {
                    if (i > 0 && s !== undefined) {
                        // use the first defined group,
                        // after the first, as a header
                        currentHeaders.push(s.trim());
                    }
                });
                return;
            }

            // normal line? push to current slide
            currentLines.push(line);
        }

        const slides = [];
        const currentLines = [];
        const currentHeaders = [];
        let currentStartOffset = 0;
        let startOffset = 0, endOffset = 0;

        for (const chr of text) {
            endOffset++;
            if (chr == "\n") {
                parseLine();
                startOffset = endOffset + 1;
            }
        }

        parseLine(); // last, unended line
        addCurrent();

        if (measure) {
            // measure each slide
            slides.forEach(s => s.measure());
        }

        return slides;
    }
}

class Deck {
    static All = [];
    static ByDay = {};

    static refresh(callback) {
        ajax({path:"/decks", success: (data) => {
            const items = (data || []).map(d => new Deck(d));
            Deck.All.splice(0, Deck.All.length, ...items);
            Deck.ByDay = {};
            items.forEach(d => Deck.ByDay[d.title] = d);

            if (callback) {
                callback.apply(this, [this]);
            }
            Recent.update();
        }});
    }

    /** create a new draft with the specified title */
    static draft(title) {
        const deck = new Deck({title:title, draft:true});
        if (!Deck.ByDay[deck.title]) {
            Deck.All.push(this);
            Deck.ByDay[deck.title] = this;
        }
        return deck;
    }

    constructor(item) {
        if (!item.title) throw "No title";
        this.title = item.title;
        this.songs = item.songs || [];
        this.text = item.text || '';

        this.initialText = this.text;
        this.slidesText = '';
        this.slides = [];
        this.loaded = false;
        this.draft = !!item.draft;
        this.dirty = false;
        this.created = null;
        this.modified = null;
    }
    get date() {
        return dayjs(this.title);
    }
    get link() {
        return 'screen.html?title='+encodeURIComponent(this.title);
    }
    load(callback) {
        if (this.loaded) {
            if (callback) callback.apply(this, [this]);
            return;
        }

        ajax({path:"/deck", qs:{title:this.title}, success:(data) => {
            this.loaded = true;
            this.dirty = false;
            this.draft = false;
            this.created = dayjs(data.created);
            this.modified = dayjs(data.modified);

            this.text = data.text;
            this.initialText = data.text;
            this.update();
            if (callback) callback.apply(this, [this]);

        }, failed:(data, req) => {
            // doesn't exist
            this.loaded = true;
            if (callback) callback.apply(this, [this]);
        }});
    }
    save(callback) {
        if (!this.dirty) return;
        const data = {title:this.title, text:this.text};
        ajax({method:'PUT', data:data, path:'/deck', success:()=> {
            this.dirty = false;
            this.draft = false;
            showMessage({msg:`saved "${this.title}`});
            if (callback) callback.apply(this, [this]);
        }});
    }
    revert() {
        if (!this.dirty) return;
        this.text = this.initialText;
        this.dirty = false;
        this.update();
    }
    delete(callback) {
        if (this.draft) {
            // not saved yet; just remove from the list
            delete Deck.ByDay[this.title];
            Deck.All.splice(Deck.All.indexOf(this), 1);
            if (callback) callback.apply(this, [this]);
            return;
        }

        ajax({method:'DELETE', path:"/deck", qs:{title:this.title}, success:(data) => {
            this.dirty = false;
            showMessage({msg:`deleted "${this.title}"`});
            if (callback) callback.apply(this, [this]);
        }});
    }

    /** update the slides to match the text */
    update() {
        if (!this.dirty && this.text != this.initialText) {
            this.dirty = true;
        }
        if (this.slidesText != this.text) {
            this.slides = Slide.Parse(this.text, true);
            this.slidesText = this.text;
        }
    }
}