// add css rules for i-name for all svg icons
// icons from: https://fonts.google.com/icons?selected=Material+Icons
(function(names) {
	const rules = names.split(',').map(n => '.i-'+n+'{background-image:url('+n+'.svg)}')
	const style = document.createElement('style');
	style.innerHTML = rules.join('\n');
	document.head.appendChild(style);
})('add,broom,clock,close,copy,discard,edit,forward,hide,imported,refresh,remote,save,screen,trash');

// dayjs docs: https://day.js.org/docs/en/parse/parse
const VueCalendar = {
	template: `#vue-calendar`,
	props: ['delta', 'used', 'state'],
	data() {
		return {
			title: "",
			headers: [],
			rows: [],
		};
	},
	watch: {
		state() {
			// update used days
			this.rows.forEach(row => {
				row.forEach(td => {
					td.used = !!this.used[td.iso];
				});
			});
		}
	},
	methods: {
		edit(title) {
			// find the deck by name, or create a new one
			const deck = Deck.ByDay[title];
			if (deck) editDeck(deck);
			else editDeck(Deck.draft(title));
		}
	},
	mounted() {
		const now = date();
		const monthStart = now.plusMonths(this.delta || 0).monthStart;

		// header
		this.title = monthStart.monthName + " " + monthStart.year;

		const today = now.iso;
		const m = monthStart.month;

		// first line
		const day = monthStart.weekStart();
		const rows = 6,
		cells = 7 * rows;
		let row = null;
		for (let i = 0; i < cells; i++) {
			if (i % 7 == 0) {
				row = [];
				this.rows.push(row);
				row.push({
					day: day,
					tx: day.weekNumber,
					cl: "weeknumber",
				});
			}

			const used = !!this.used[day.iso];
			const td = { tx: day.day, iso: day.iso, cl: ["day"], used: used };
			row.push(td);

			const current = day.month == m;
			td.cl.push("day-" + (current ? "current" : "other"));

			if (i % 7 == 5 || i % 7 == 6) {
				td.cl.push("weekend");
			}

			if (td.iso == today) td.cl.push("today");
			day.moveDays(1);
		}
	}
};

const VueThumbs = {
	template: `#vue-thumbs`,
	props: ['slides', 'selected', 'clickable', 'editor'],
};

const tabs = new Tabs();

function editDeck(deck) {
	const found = tabs.find('editor', deck.title);
	if (found) {
		found.show();
		return;
	}

	tabs.add({kind:'editor', title:deck.title, icon:'edit', app:{
		template: '#vue-editor',
		components: {thumbs:VueThumbs},
		data() {
			return {
				deck: deck,
				thumb: null,
				initialText: deck.text,
			}
		},
		mounted() {
			this.deck.load(() => {
				if (this.tab.active && this.tab.vue) {
					this.tab.vue.$refs.editor.focus();
				}
			});
		},
		methods: {
			trash() {
				const t = this.deck.title;
				if (!confirm('Really delete "' + t + '"?')) return;
				ajax({method:'DELETE', path:"/deck", qs:{title:t}, success:(data) => {
					showMessage({msg:'deleted "'+t+'"'});
					this.deck.dirty = false;
					this.tab.close();

					// update list
					// const idx = decks_tab.vue.decks.findIndex(d => d.title == this.deck.title);
					// if (idx != -1) decks_tab.vue.decks.splice(idx, 1);
				}});
			},
			show(slide) {
				showContent(this.deck.title, slide.text);
				this.thumb = slide;
			},
			addText(text) {
				this.deck.dirty = true;
				this.deck.text += '\n' + text + '\n';
			}
		}
	}, onfocus:(tab)=> {
		if (tab.vue) {
			tab.vue.$refs.editor.focus();
		}
	}}).show();
}

function presentDeck(deck) {
	const found = tabs.find('remote', deck.title);
	if (found) {
		found.show();
		return;
	}

	const tab = tabs.add({kind:'remote', title:deck.title, icon:'remote', app:{
		template: '#vue-remote',
		components: {thumbs:VueThumbs},
		data() {
			return {
				deck: deck,
				thumb: null
			}
		},
		mounted() {
			this.deck.load();
		},
		methods: {
			hide() {
				showContent(deck.title, '');
			},
			show(slide) {
				showContent(deck.title, slide.text);
				this.thumb = slide;
			},
		}
	}});
	tab.show();
}

function showContent(title, content) {
	ajax({method:"POST", path:"/show", data:{title:title, show:content}});
}

const decks_tab = tabs.add({kind:'decks', title:'Decks', app:{
	template: '#vue-decks',
	data() {
		return {
			refreshCount: 0,
			decks: Deck.All,
			recent: Recent,
			showRecent: false,
			used: Deck.ByDay
		}
	},
	mounted() {
		this.refresh();
	},
	components: {
		calendar: VueCalendar,
	},
	methods: {
		refresh() {
			Deck.refresh(() => {
				this.refreshCount++;
			});
		},
		edit(deck) { editDeck(deck); },
		present(deck) { presentDeck(deck); }
	}
}});

function match(search, norm, text) {
	if (norm.length != text.length || search.length > norm.length) {
		// no possible match
		return null;
	}

	const word_start_score = 3;
	const middle_score = 1;
	const word_score = 6;
	const full_match = 12;

	let si = 0, score = 0;
	let markup = '';
	let word_start = true;
	let prev_char_matched = false;

	for (let ni = 0, nl = norm.length; ni < nl; ni++) {
		const searchchar = si < search.length ? search[si] : '';
		const normchar = norm[ni];
		const textchar = text[ni];

		if (normchar == ' ') {
			// text is whitespace, skip
			markup += textchar;
			word_start = true;
			continue;
		}

		if (normchar == searchchar) {
			// text matches search
			if (!prev_char_matched) {
				prev_char_matched = true;
				markup += '<ins>';
			}
			score += word_start ? word_start_score : middle_score;
			si++;

		} else {
			if (prev_char_matched) {
				prev_char_matched = false;
				markup += '</ins>';
			}
		}
		markup += textchar;
		word_start = false;
	}

	if (si != search.length) {
		// not a full match
		return null;
	}

	if (prev_char_matched) {
		// end pending markup
		markup += '</ins>';
	}

	if (norm.indexOf(search) != -1) {
		// whole word found
		score += (norm.length == search.length) ? full_match : word_score;
	}
	return {
		score: score,
		markup: markup
	};
}

const songs_tab = tabs.add({kind:'songs', title:'Songs', app:{
	template: '#vue-songs',
	data() {
		return {
			songs: Song.All,
			matches: [],
			selected: null,
			search_text: ''
		}
	},
	created() {
		this.refresh();
	},
	mounted() {
		if (this.tab.active && this.tab.vue) {
			this.tab.vue.$refs.search.focus();
		}
	},
	methods: {
		refresh() {
			Song.refresh(() => {
				this.search();
			});
		},
		search() {
			this.selected = null;
			const norm = normalize(this.search_text.trim()).replace(/\s/g, '');
			if (!norm) {
				this.matches = this.songs.map(s => ({song:s}));
				return;
			}

			// do a fuzzy search on song titles
			const matches = [];
			this.songs.forEach((s) => {
				const m = match(norm, normalize(s.title), s.title);
				if (m != null) {
					if (!s.imported) m.score += 2; // prefer non-imported songs
					matches.push(extend(m, {song:s}));
				}
			});
			matches.sort((a, b) => a.score < b.score);
			this.matches = matches;
		},
		/** adds the current song to either the last selected editor */
		addToEditor() {
			if (!this.selected) return; // no selected song
			let tab = this.tabs.previous;
			if (!tab || tab.kind != 'editor') {
				const editors = tabs.items.filter(t => t.kind == 'editor');
				if (editors.length != 1) return; // multiple or no editor
				tab = editors[0]; // only one editor; use it
			}

			// load the song
			const song = this.selected.song;
			song.load((song) => {
				if (!song.text) {
					showMessage({kind:'error', msg:`No text for "${song.title}"...`});
					return;
				}

				tab.vue.addText(song.pasteText);
				tab.show();
				showMessage({msg:`Added "${song.title}" to ${tab.title}`});
			});
		},
		clean() {
			this.search_text = '';
			this.search();
		},
		copy() {
			ajax({path:"/song", qs:{song_id:this.selected.song.id}, success: (data) => {
				navigator.clipboard.writeText(data.text).then(() => {
					showMessage({msg:'Copied to the Clipboard!'});
				});
			}});
		},
		edit(selected) {
			let song = selected ? selected.song : Song.draft();
			if (selected) {
				const found = tabs.items.find(t => t.kind == 'song' && t.vue.song.id == song.id);
				if (found) {
					// there's already an editor open for this song
					found.show();
					return;
				}
			}

			// edit or create a new song
			tabs.add({kind:'song', title:'Loading...', icon:'edit', app:{
				template: '#vue-song',
				components: {thumbs:VueThumbs},
				data() {
					return {song: song}
				},
				mounted() {
					this.refresh();
				},
				methods: {
					refresh() {
						this.song.load(() => {
							this.update();
							this.focus();
						});
					},
					trash() {
						if (!confirm(`Really delete "${this.song.title}"?`)) return;
						ajax({method:'DELETE', path:"/song", qs:{song_id:this.song.id}, success:(data) => {
							showMessage({msg:`deleted "${this.song.title}"`});
							this.song.dirty = false;
							this.tab.close();

							// update list
							const idx = songs_tab.vue.songs.findIndex(s => s.id == this.song.id);
							if (idx != -1) songs_tab.vue.songs.splice(idx, 1);
						}});
					},
					cleanup() {
						const text = Slide.Cleanup(this.$refs.editor.value);
						if (this.song.text != text) {
							this.song.text = text;
							this.song.dirty = true;
						}
					},
					focus() {
						if (!(this.tab.active && this.tab.vue)) {
							this.tab.show();
						}

						const tryFocus = () => {
							if (this.tab.active && this.tab.vue) {
								this.tab.vue.$refs.editor.focus();
								return;
							}
							setTimeout(() => tryFocus(), 10);
						};
						tryFocus();
					},
					save() {
						if (!this.song.dirty) return;

						// extract title from text before saving
						if (this.song.text.match(/^\s*#/)) {
							const lines = this.song.text.split('\n');
							this.song.title = lines[0].replace(/^\s*#\s*/, '');
							this.song.text = lines.slice(1).join('\n');
						}

						// create/update
						const method = this.song.id ? 'PUT' : 'POST';
						ajax({method:method, data:this.song, path:'/song', success:(song)=> {
							this.load(song);
							showMessage({msg:`saved "${this.song.title}"`});

							// update the list
							let found = false;
							for (let i = 0, l = songs_tab.vue.songs.length; i < l; i++) {
								const s = songs_tab.vue.songs[i];
								if (s.id == song.id) {
									s.update({title:song.title});
									found = true;
									break;
								}
							}
							if (!found) {
								songs_tab.vue.songs.push(new Song(song));
							}

							// re-run search
							songs_tab.vue.search();

						}, failed: ()=> {
							this.load(this.song);
							showMessage({kind:'error', msg:'error saving '+this.song.title});
						}});
					},
					update() {
						this.song.update();

						// update title
						const title = this.song.title + (this.song.id ? ` (@${this.song.id})` : '');
						if (this.tab.title != title) {
							this.tab.title = title;
						}
					},
					keyup() {
						if (this.song.text == this.slidesText) return;
						this.song.dirty = true;
						this.update();
					},
					show(thumb) {
						if (thumb.label) return;
						showContent(this.song.title, thumb.text)
					}
				}
			}});
		}
	}
}, onfocus:(tab)=> {
	if (tab.vue) {
		tab.vue.$refs.search.focus();
	}
}});

tabs.add({kind:'help', title:'Help', app:{
	template: '#vue-help',
}});

// save on Ctrl+S/Cmd+S
document.addEventListener('keydown', function(e) {
	let handled = false;
	if (e.composed && e.key == 's' && (e.ctrlKey || e.metaKey)) {
		const tab = tabs.active;
		if (tab) {
			switch (tab.kind) {
				case 'editor':
					handled = true;
					tab.vue.deck.save();
					break;
				case 'song':
					handled = true;
					tab.vue.song.save();
					break;
			}
		}
	}
	if (handled) {
		e.stopImmediatePropagation();
		e.preventDefault();
	}
});

ajax({method:'GET', path:'/version', success:(version)=> {
	// show version
	document.querySelector('#version').innerText = version || 'no-version';
}});
