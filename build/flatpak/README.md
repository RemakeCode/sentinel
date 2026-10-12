# Flatpak desktop build

The Flatpak keeps `org.gnome.Platform//50` for Sentinel's GTK/WebKit desktop and
bundles Qt base 6.10.2 and LayerShellQt 6.6.0 for custom popups. Both notification
bundles are built inside the SDK before Wails bindings and Go compilation;
generated assets from a host checkout are excluded.

A build-only LayerShellQt patch omits its QML bindings and demo; the helper uses
the Widgets interface and Wayland shell plugin.

On Linux with Flatpak, flatpak-builder, and the Flathub user remote configured:

```sh
wails3 task linux:create:flatpak
flatpak install --user ./bin/sentinel-<version>-<architecture>.flatpak
flatpak run io.github.remakecode.sentinel
```

The bundle uses the build machine's architecture (`x86_64` or `aarch64`). Qt
libraries and Wayland plugins live inside the Flatpak, so users do not need host
Qt packages.

GNOME setup uses a scoped writable mount of the host's
`xdg-data/gnome-shell/extensions/io.github.remakecode.sentinel` directory and
D-Bus access to `org.gnome.Shell`. The existing installer can use its XDG data
path through that mount, including a custom host XDG data root. It refuses to
install into an unmounted app-private directory. A newly installed extension
still needs logout/login for Shell discovery.

Check Native delivery, then Custom earned/progress/rare notifications and sound
on GNOME and a layer-shell desktop. Keep the terminal open to see helper library
or plugin errors and GNOME setup/delivery errors.
