# Sentinel GNOME notifications

The extension source is TypeScript. Desktop builds compile it to JavaScript and copy its stylesheet, metadata, and Sentinel artwork into `backend/notifier/out/io.github.remakecode.sentinel/`. `backend/notifier/custom_notification_wails.go` embeds that directory together with the Qt helper. Selecting **Custom** notifications on GNOME Wayland installs or updates it in `$XDG_DATA_HOME/gnome-shell/extensions/io.github.remakecode.sentinel/`, normally `~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel/`. Startup with Custom already selected also installs updates and restores missing files. Users do not need Node or TypeScript installed. Decky builds exclude these assets.

Target Shell versions are 48–50, on x86-64 or ARM64. The former standalone sample was used on a GNOME live session; the integrated extension and animated rare glow still need GNOME host acceptance. Flatpak host installation and runtime packaging remain deferred; sandbox installation currently reports an error instead of writing to an app-private extension directory.

## Enable after installation

Open GNOME's **Extensions** app and enable **sentinel custom notification**, or run:

```sh
gnome-extensions enable io.github.remakecode.sentinel
```

If Shell says the extension does not exist, log out and back in so it discovers the newly installed files, then enable it. Ordinary enabling/disabling of a discovered extension does not need logout. When Sentinel updates its JavaScript, start a fresh Shell session to load the new code. Enabling registers the endpoint; use Sentinel Settings' **Test Notification** or progress test to display a card.

If you installed an earlier version, disable its old UUID with `gnome-extensions disable sentinel@remakecode` (or `sentinel-sample@remakecode` for the original sample) before enabling the new UUID shown above.

## Layout and delivery

Go's existing queue owns eligibility, sequence, sound selection, and duration. `backend/notifier/custom_notification.go` selects the GNOME sender on a GNOME/Mutter Wayland session and sends the same JSON fields as the Qt helper to the session bus:

- Destination: `org.gnome.Shell`
- Object: `/io/github/remakecode/Sentinel/Notifications`
- Method: `io.github.remakecode.Sentinel.Notifications.Show`
- Input: one JSON string; a successful method return means the request was accepted.

The extension displays plain text, game header, artwork with bundled fallback, and counts above an 8px orange progress bar. Rare earned achievements show a soft gold ring whose highlight rotates once every three seconds; progress never animates rarity. Actors do not request focus or pointer input. Disabling removes the endpoint, card, lifetime timer, animation, and monitor signal. Errors do not trigger Qt or native fallback. Sound plays in Go after the extension accepts the request.

A new request replaces the current card. The extension uses the supplied `durationMs`, including temporary backend timing changes for visual work. The examples below explicitly expire after seven seconds and 4.9 seconds.

## Manual development

From the project root on a GNOME host:

```sh
wails3 task common:build:gnome-extension
mkdir -p ~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel
cp backend/notifier/out/io.github.remakecode.sentinel/* \
  ~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel/
```

For a custom `XDG_DATA_HOME`, use its `gnome-shell/extensions/io.github.remakecode.sentinel/` directory instead. Start a fresh Shell session if needed, then enable the extension. Manual changes to installed files will be replaced by the app's bundled version on its next setup; edit the project sources and rebuild Sentinel to update the bundle.

Earned notification:

```sh
gdbus call --session \
  --dest org.gnome.Shell \
  --object-path /io/github/remakecode/Sentinel/Notifications \
  --method io.github.remakecode.Sentinel.Notifications.Show \
  '{"title":"For those who come after","description":"Discover all hidden locations","gameName":"Sentinel","iconPath":"","progress":0,"maxProgress":0,"isProgress":false,"isRare":false,"durationMs":7000}'
```

Progress notification:

```sh
gdbus call --session \
  --dest org.gnome.Shell \
  --object-path /io/github/remakecode/Sentinel/Notifications \
  --method io.github.remakecode.Sentinel.Notifications.Show \
  '{"title":"For those who come after","description":"Discover all hidden locations","gameName":"Sentinel","iconPath":"","progress":7,"maxProgress":10,"isProgress":true,"isRare":false,"durationMs":4900}'
```

Set `isRare` to `true` on the earned example to try the animated glow. Set `iconPath` to an absolute image path for custom artwork. Empty or unreadable artwork uses the bundled Sentinel icon. There is no preview flag or separate rare test in Settings.

## Edit and debug

Edit `io.github.remakecode.sentinel/extension.ts` for layout and `stylesheet.css` for appearance. `task dev` watches TypeScript, CSS, and metadata; desktop builds compile and embed the changes automatically. Generated files are ignored by Git. For a direct Go build or binding generation, run `wails3 task common:build:gnome-extension` first, along with the Qt helper build. Shell still needs a fresh session after installed JavaScript changes. GNOME 49/50 support `dbus-run-session gnome-shell --devkit --wayland`; GNOME 48 uses `--nested` instead of `--devkit`. Run enable and notification commands inside that nested session.

The TypeScript setup uses the GNOME type definitions described in the [GJS TypeScript guide](https://gjs.guide/extensions/development/typescript.html).

```sh
journalctl -f -o cat /usr/bin/gnome-shell
gnome-extensions disable io.github.remakecode.sentinel
```

GNOME 50 removed `affectsInputRegion`, so the extension supplies it only on 48/49. An old sample passing it on 50 could display a card and throw before starting its expiry timer. The integrated handler cleans up its card if creating a notification throws.

References: [GNOME extension development](https://gjs.guide/extensions/development/creating.html), [GJS D-Bus services](https://gjs.guide/guides/gio/dbus.html), [GNOME 48 orientation API](https://gjs.guide/extensions/upgrading/gnome-shell-48.html).
