# Slides

This is a **simple** slide deck build and presenter, focused on song lyrics.

It's specially useful for using remotely with Zoom or OBS, since the presenter
and screen may be on different computers/networks.

## Editor view

Simply write or paste text on the editor, to show slides on the preview area.
Text is converted into slides, which are shown in a preview area at the right.
One or more empty lines start a new slide, Markdown-style.

Lines that start with "#", or that look like
song section names ("intro", "chorus", "verse", etc) are converted to titles.
Chords and markings such as "(2x)" are automagically ignored.
Usually, pasting lyrics with chords mixed in Just Works™️.

All changes are saved automatically.

Clicking a slide on the preview area will set that as the current
content of the screen. Clicking on "clear screen" will clear it.

The preview area scrolls automatically to the focused paragraph.

![editor](attic/sample-editor.jpg)

## Screen view

The screen view will be updated with the current content.
It's possible to use screens and decks independently, and different
users can open the same deck simultaneously.

For instance, user A can use change the current slide on the editor,
and user B viewing a screen on a different network
will see the change instantly.

![screen](attic/sample-screen.jpg)

### Building/running

Build it and create a `config.yaml`, containing the address to serve on,
the base URL, and paths to the data directory
and the directory containing static files.

```
address: localhost:8080
baseurl: //localhost:8080
path:
  static: ./static
  data: ./data
```

Running it with `-dev` will disable cache for static resources,
and reload them on each pageview.
