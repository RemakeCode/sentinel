# Sentinel GNOME notifications

The extension source is TypeScript. Desktop builds compile it to JavaScript and copy its stylesheet, metadata, and Sentinel artwork into `backend/notifier/out/io.github.remakecode.sentinel/`. Sentinel embeds these files in the desktop application. Selecting **Custom** notifications on GNOME Wayland installs or updates it in `$XDG_DATA_HOME/gnome-shell/extensions/io.github.remakecode.sentinel/`, normally `~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel/`. Startup with Custom already selected also installs updates and restores missing files. Users do not need Node or TypeScript installed. Decky builds exclude these assets.

Target Shell versions are 48–50, on x86-64. The extension and animated rare glow still need GNOME host acceptance. Flatpak host installation and runtime packaging remain deferred; sandbox installation currently reports an error instead of writing to an app-private extension directory.

## Enable after installation

After Sentinel first installs the extension, **log out and log back in**. GNOME Shell will not discover the newly copied extension during the current session.

Then open GNOME's **Extensions** app and enable **sentinel custom notification**, or run:

```sh
gnome-extensions enable io.github.remakecode.sentinel
```

Once discovered, ordinary enabling/disabling does not need logout. Enabling registers the endpoint; use Sentinel Settings' **Test Notification** or progress test to display a card.

## Layout and delivery

Go's existing queue owns eligibility, sequence, sound selection, and duration. On GNOME Wayland, Sentinel sends notification data to the extension over the session bus:

- Destination: `org.gnome.Shell`
- Object: `/io/github/remakecode/Sentinel/Notifications`
- Method: `io.github.remakecode.Sentinel.Notifications.Show`
- Input: one JSON string; a successful method return means the request was accepted.

The extension displays plain text, game header, artwork with bundled fallback, and counts above an 8px orange progress bar. Rare earned achievements show a soft gold ring whose highlight rotates once every three seconds; progress never animates rarity. Actors do not request focus or pointer input. Disabling removes the endpoint, card, lifetime timer, animation, and monitor signal. Delivery failures are logged without fallback. Sound plays in Go after the extension accepts the request.

A new request replaces the current card. The extension uses the supplied `durationMs`, including temporary backend timing changes for visual work. The examples below explicitly expire after seven seconds and 4.9 seconds.

## Manual development

From the project root on a GNOME host:

```sh
wails3 task common:build:gnome-extension
mkdir -p ~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel
cp backend/notifier/out/io.github.remakecode.sentinel/* \
  ~/.local/share/gnome-shell/extensions/io.github.remakecode.sentinel/
```

For a custom `XDG_DATA_HOME`, use its `gnome-shell/extensions/io.github.remakecode.sentinel/` directory instead. After copying the files, log out and back in, then enable the extension. Manual changes to installed files will be replaced by the app's bundled version on its next setup; edit the project sources and rebuild Sentinel to update the bundle.

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

For development, edit `io.github.remakecode.sentinel/extension.ts` for layout and `stylesheet.css` for appearance. Sentinel's desktop build compiles and bundles the extension automatically; users do not build it separately.

The TypeScript setup uses the GNOME type definitions described in the [GJS TypeScript guide](https://gjs.guide/extensions/development/typescript.html).

```sh
journalctl -f -o cat /usr/bin/gnome-shell
gnome-extensions disable io.github.remakecode.sentinel
```

GNOME 50 removed `affectsInputRegion`, so the extension supplies it only on 48/49. The handler cleans up its card if creating a notification throws.

References: [GNOME extension development](https://gjs.guide/extensions/development/creating.html), [GJS D-Bus services](https://gjs.guide/guides/gio/dbus.html), [GNOME 48 orientation API](https://gjs.guide/extensions/upgrading/gnome-shell-48.html).
