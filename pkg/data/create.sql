create table if not exists users (
	rowid integer primary key,
	username text not null unique,
	name text,
	passwd text
);

create table if not exists decks (
	rowid integer primary key,
	title text not null unique,
	text text,
	current text,
	creator text,
	lastmod text,
	created datetime default current_timestamp,
	modified datetime default current_timestamp
);

create table if not exists songs (
	rowid integer primary key,
	external_id text unique,
	title text not null,
	author text,
	ccli text,
	content text,
	created datetime default current_timestamp,
	modified datetime default current_timestamp
);

-- ensure we have a system user
insert or ignore into users (username, name)
values ('system', 'System');
