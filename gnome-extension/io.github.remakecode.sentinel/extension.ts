import Clutter from 'gi://Clutter';
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import Pango from 'gi://Pango';
import St from 'gi://St';
import Cairo from 'cairo';

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import {PACKAGE_VERSION} from 'resource:///org/gnome/shell/misc/config.js';

interface NotificationData {
    title: string;
    description: string;
    gameName: string;
    iconPath: string;
    progress: number;
    maxProgress: number;
    isProgress: boolean;
    isRare: boolean;
    durationMs: number;
}

const INTERFACE = `
<node>
  <interface name="io.github.remakecode.Sentinel.Notifications">
    <method name="Show">
      <arg name="notification" type="s" direction="in"/>
    </method>
  </interface>
</node>`;

// Draw a stationary rounded ring. Only the highlight moves, so corners never protrude.
function paintRareGlow(area: St.DrawingArea, phase: number) {
    const cr = area.get_context();
    const [width, height] = area.get_surface_size();
    const scale = St.ThemeContext.get_for_stage(global.stage).scale_factor;
    const inset = 10 * scale;
    const radius = 11 * scale;
    const angle = phase * Math.PI * 2 - Math.PI / 2;
    const dx = Math.cos(angle) * width / 2;
    const dy = Math.sin(angle) * height / 2;

    cr.arc(width - inset - radius, inset + radius, radius, -Math.PI / 2, 0);
    cr.arc(width - inset - radius, height - inset - radius, radius, 0, Math.PI / 2);
    cr.arc(inset + radius, height - inset - radius, radius, Math.PI / 2, Math.PI);
    cr.arc(inset + radius, inset + radius, radius, Math.PI, Math.PI * 1.5);
    cr.closePath();

    // Overlapping translucent strokes soften the ring without blurring the artwork.
    for (const [lineWidth, opacity] of [[12, 0.05], [8, 0.1], [5, 0.2], [2, 0.7]]) {
        const gradient = new Cairo.LinearGradient(
            width / 2 - dx, height / 2 - dy, width / 2 + dx, height / 2 + dy);
        gradient.addColorStopRGBA(0, 1, 0.68, 0, 0);
        gradient.addColorStopRGBA(0.55, 1, 0.68, 0, 0);
        gradient.addColorStopRGBA(0.8, 1, 0.84, 0.3, opacity * 0.5);
        gradient.addColorStopRGBA(1, 1, 0.96, 0.7, opacity);
        cr.setSource(gradient);
        cr.setLineWidth(lineWidth * scale);
        cr.strokePreserve();
    }
    cr.$dispose();
}

export default class SentinelNotifications extends Extension {
    private _card: St.BoxLayout | null = null;
    private _timeout = 0;
    private _glowTimeline: Clutter.Timeline | null = null;
    private _dbus: Gio.DBusExportedObject | null = null;
    private _monitorsChanged = 0;

    enable() {
        this._card = null;
        this._timeout = 0;
        this._glowTimeline = null;
        this._dbus = Gio.DBusExportedObject.wrapJSObject(INTERFACE, this);
        this._dbus.export(Gio.DBus.session, '/io/github/remakecode/Sentinel/Notifications');
        this._monitorsChanged = Main.layoutManager.connect('monitors-changed', () => {
            this._clear();
        });
    }

    disable() {
        this._dbus?.unexport();
        this._dbus = null;
        Main.layoutManager.disconnect(this._monitorsChanged);
        this._monitorsChanged = 0;
        this._clear();
    }

    // Same JSON fields as the Qt helper. A new request replaces the current card.
    Show(json: string) {
        this._clear();
        try {
            this._show(JSON.parse(json));
        } catch (error) {
            this._clear();
            throw error;
        }
    }

    private _show(data: NotificationData) {
        const card = new St.BoxLayout({
            style_class: 'sentinel-card',
            orientation: Clutter.Orientation.VERTICAL,
            reactive: false,
            can_focus: false,
        });
        this._card = card;

        const gameName = new St.Label({
            text: data.gameName,
            style_class: 'sentinel-game-name',
        });
        card.add_child(gameName);

        const row = new St.BoxLayout({style_class: 'sentinel-row'});
        card.add_child(row);

        const defaultIcon = new Gio.FileIcon({file: this.dir.get_child('sentinel.png')});
        const icon = new St.Icon({
            gicon: data.iconPath
                ? new Gio.FileIcon({file: Gio.File.new_for_path(data.iconPath)})
                : defaultIcon,
            fallback_gicon: defaultIcon,
            icon_size: 80,
            x_align: Clutter.ActorAlign.CENTER,
            y_align: Clutter.ActorAlign.CENTER,
            style_class: 'sentinel-icon',
        });
        if (data.isRare && !data.isProgress) {
            icon.add_style_class_name('sentinel-rare');
            const layers = new St.Widget({
                style_class: 'sentinel-icon-layers',
                layout_manager: new Clutter.BinLayout(),
                y_align: Clutter.ActorAlign.CENTER,
            });
            const glow = new St.DrawingArea({x_expand: true, y_expand: true});
            let phase = 0;
            glow.connect('repaint', area => paintRareGlow(area, phase));
            layers.add_child(glow);
            layers.add_child(icon);
            row.add_child(layers);
            this._glowTimeline = new Clutter.Timeline({
                actor: glow,
                duration: 3000,
                repeat_count: -1,
            });
            this._glowTimeline.connect('new-frame', timeline => {
                phase = timeline.get_progress();
                glow.queue_repaint();
            });
        } else {
            row.add_child(icon);
        }

        const content = new St.BoxLayout({
            orientation: Clutter.Orientation.VERTICAL,
            style_class: 'sentinel-content',
            x_expand: true,
            y_align: Clutter.ActorAlign.CENTER,
        });
        const contentColumn = new St.Bin({
            style_class: 'sentinel-content-column',
            child: content,
            x_expand: true,
            y_align: Clutter.ActorAlign.CENTER,
        });
        row.add_child(contentColumn);

        const title = new St.Label({text: data.title, style_class: 'sentinel-title'});
        title.clutter_text.line_wrap = true;
        title.clutter_text.ellipsize = Pango.EllipsizeMode.NONE;
        content.add_child(title);

        const description = new St.Label({
            text: data.description,
            style_class: 'sentinel-description',
        });
        description.clutter_text.line_wrap = true;
        description.clutter_text.ellipsize = Pango.EllipsizeMode.NONE;
        content.add_child(description);

        if (data.isProgress) {
            const progressDetails = new St.BoxLayout({
                orientation: Clutter.Orientation.VERTICAL,
                style_class: 'sentinel-progress-details',
            });
            content.add_child(progressDetails);
            const progressCount = new St.Label({
                text: `${data.progress}/${data.maxProgress}`,
                style_class: 'sentinel-progress-count',
                x_align: Clutter.ActorAlign.END,
            });
            progressDetails.add_child(progressCount);

            const track = new St.BoxLayout({
                style_class: 'sentinel-progress-track',
                orientation: Clutter.Orientation.HORIZONTAL,
                x_expand: true,
            });
            const fill = new St.Widget({
                style_class: 'sentinel-progress-fill',
                x_align: Clutter.ActorAlign.START,
                y_expand: true,
            });
            track.add_child(fill);
            track.add_child(new St.Widget({x_expand: true}));
            const ratio = data.maxProgress > 0
                ? Math.max(0, Math.min(1, data.progress / data.maxProgress))
                : 0;
            track.connect('notify::width', () => {
                fill.width = Math.round(track.width * ratio);
            });
            progressDetails.add_child(track);
        }

        const monitor = Main.layoutManager.currentMonitor;
        if (!monitor)
            throw new Error('No monitor available for the notification');

        const workArea = Main.layoutManager.getWorkAreaForMonitor(monitor.index);
        const scale = St.ThemeContext.get_for_stage(global.stage).scale_factor;
        const position = () => card.set_position(
            workArea.x + workArea.width - card.width - 24 * scale,
            workArea.y + workArea.height - card.height - 24 * scale
        );
        card.connect('notify::width', position);
        card.connect('notify::height', position);
        // GNOME 50 removed the input-region option; actors remain nonreactive.
        Main.layoutManager.addTopChrome(card,
            Number.parseInt(PACKAGE_VERSION, 10) < 50 ? {affectsInputRegion: false} : {});
        position();
        this._glowTimeline?.start();

        this._timeout = GLib.timeout_add(GLib.PRIORITY_DEFAULT, data.durationMs, () => {
            this._timeout = 0;
            this._clear();
            return GLib.SOURCE_REMOVE;
        });
    }

    private _clear() {
        this._glowTimeline?.stop();
        this._glowTimeline = null;

        if (this._timeout) {
            GLib.Source.remove(this._timeout);
            this._timeout = 0;
        }

        if (this._card) {
            Main.layoutManager.removeChrome(this._card);
            this._card.destroy();
            this._card = null;
        }
    }
}
