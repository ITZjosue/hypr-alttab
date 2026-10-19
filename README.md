# hypr-dock
### Interactive dock panel and Alt+Tab switcher for Hyprland


This repository ships two programs:
- **`hypr-dock`** - the dock panel
- **`hypr-alttab`** - an Alt+Tab window/workspace switcher with live previews (see [hypr-alttab](#hypr-alttab))

> Fork of [lotos-linux/hypr-dock](https://github.com/lotos-linux/hypr-dock).

<img width="1360" height="768" alt="250725_16h02m52s_screenshot" src="https://github.com/user-attachments/assets/041d2cf6-13ba-4c89-a960-1903073ff2d4" />
<img width="1360" height="768" alt="250725_16h03m09s_screenshot" src="https://github.com/user-attachments/assets/0c1ad8ca-37c1-4fd6-a48d-46f74c2d2609" />

[![YouTube](https://img.shields.io/badge/YouTube-Video-FF0000?logo=youtube)](https://youtu.be/HHUZWHfNAl0?si=ZrRv2ggnPBEBS5oY)

## Installation

### Dependencies

- `go` (make)
- `gtk3`
- `gtk-layer-shell`

### Installation
! The first build may take a very long time due to gtk3 bindings !
```bash
git clone https://github.com/ITZjosue/hypr-alttab.git
cd hypr-alttab
make get
make build
make install
```
`make build` and `make install` cover both programs. To work on one of them only:
```bash
make build-dock     && make install-dock
make build-alttab   && make install-alttab
```
The install targets stop the running instance, so the next launch picks up the new binary.

### Uninstallation
```bash
make uninstall
```

### Local run (dev mode)
```bash
make exec
```

## Running

### Launch parameters:

```text
  -config string
    	config file (default "~/.config/hypr-dock")
  -dev
    	enable developer mode
  -log-level string
    	log level (default "info")
  -theme string
    	theme dir
```
#### All parameters are optional.

Default configuration and themes are installed in `/etc/hypr-dock`
On first run, they are copied to `~/.config/hypr-dock`
### Add to `hyprland.conf`:

```text
exec-once = hypr-dock
bind = Super, D, exec, hypr-dock
```

### And configure blur if needed
```text
layerrule = blur true,match:namespace hypr-dock
layerrule = ignore_alpha 0,match:namespace hypr-dock
layerrule = blur true,match:namespace dock-popup
layerrule = ignore_alpha 0,match:namespace dock-popup
```

#### The dock only supports one running instance, so running it again will close the previous one.

## Configuration

#### !Important!
If you already had `hypr-dock` installed or you are updating, please note that starting from version 1.2.0, an `ini`-like configuration format is now used. If you have old configs remaining in your user folder, be sure to delete or move them. Then simply adapt your settings to the new format.

### Available parameters in `hypr-dock.conf`

```ini
[General]
CurrentTheme = lotos

# Icon size (px) (default 23)
IconSize = 23

# Window overlay layer height (background, bottom, top, overlay) (default top)
Layer = top

# Exclusive Zone (true, false) (default true)
Exclusive = true

# SmartView (true, false) (default false)
SmartView = false

# Window position on screen (top, bottom, left, right) (default bottom)
Position = bottom

# Delay before hiding the dock (ms) (default 400)
AutoHideDelay = 400   # Only for SmartView

# Use system gap (true, false) (default true)
SystemGapUsed = true

# Indent from the edge of the screen (px) (default 8)
Margin = 8

# Distance of the context menu from the window (px) (default 5)
ContextPos = 5

[General.preview]
# Window thumbnail mode selection (none, live, static) (default none)
Mode = none
# "none"   - disabled (text menus)
# "static" - last window frame
# "live"   - window streaming
      
# !WARNING! 
# BY SETTING "Mode" TO "live" OR "static", YOU AGREE TO THE CAPTURE 
# OF WINDOW CONTENTS.
# THE "HYPR-DOCK" PROGRAM DOES NOT COLLECT, STORE, OR TRANSMIT ANY DATA.
# WINDOW CAPTURE OCCURS ONLY FOR THE DURATION OF THE THUMBNAIL DISPLAY!
#   
# Source code: https://github.com/ITZjosue/hypr-alttab

# Live preview fps (0 - ∞) (default 30)
FPS = 30

# Live preview bufferSize (1 - 20) (default 5)
BufferSize = 5

# Popup show/hide/move delays (ms)
ShowDelay = 500  # (default 500)
HideDelay = 350  # (default 350)
MoveDelay = 100  # (default 100)
```
#### If a parameter is not specified, the default value will be used

## Understanding non-obvious parameters

### SmartView
Similar to auto-hide: if `true`, the dock stays beneath all windows, but moves the cursor to the screen edge - the dock rises above them

### Exclusive
Activates special layer behavior where tiling windows do not overlap the dock

### SystemGapUsed
- When `SystemGapUsed = true`, the dock sets its margin from the screen edge using values from the `hyprland` configuration, specifically `general:gaps_out` values, and dynamically updates when the `hyprland` configuration changes
- When `SystemGapUsed = false`, the margin from the screen edge is set by the `Margin` parameter

### General.preview
- `ShowDelay`, `HideDelay`, `MoveDelay` - preview popup action delays in milliseconds
- `FPS`, `BufferSize` - only used when `Mode = live`

#### Preview appearance settings are configured through theme files

### Pinned applications are stored in `~/.local/share/hypr-dock/pinned`
To pin/unpin, open the application's context menu in the dock and click `pin`/`unpin`
#### Example
```text
firefox
code-oss
kitty
org.telegram.desktop
nemo
org.kde.ark
sublime_text
qt6ct
one.ablaze.floorp
```
You can edit it manually. But why? ¯\_(ツ)_/¯

## Themes

#### Themes are located in `~/.config/hypr-dock/themes/`

### A theme consists of
- `theme.conf`
- `style.css`
- A folder with `svg` files for indicating the number of running applications (see [themes.md](docs/customize/themes.md))

### Theme config
```ini
[Theme]
# Distance between elements (px) (default 9)
Spacing = 5


[Theme.preview]
# Size (px) (default 120)
Size = 120

# Image/Stream border-radius (px) (default 0)
BorderRadius = 0

# Popup padding (px) (default 10)
Padding = 10
```
#### Customize `style.css` as you wish. Detailed styling documentation will be provided later.

## hypr-alttab

A fullscreen-overlay Alt+Tab switcher with window previews. It can cycle either between **workspaces** (one card per workspace, showing its window layout) or between the **windows of the current workspace** (one card per window).

### Add to `hyprland.conf`

```text
bind = ALT, Tab, exec, hypr-alttab
bind = ALT SHIFT, Tab, exec, hypr-alttab
```

Do not also bind `cyclenext` to the same keys: both would fire and fight over focus.

Optional blur:
```text
layerrule = blur true,match:namespace hypr-dock-switcher
layerrule = ignore_alpha 0,match:namespace hypr-dock-switcher
```

### Controls

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | Next / previous entry |
| `←` `→` | Previous / next entry |
| `↑` `↓` | Move one row up / down |
| Release `Alt` (or `Super`) / `Enter` | Focus the selected entry |
| `Esc` | Close without switching |

### How it runs

The first launch starts a background daemon; later launches just tell it to show itself, which keeps it instant. The config is read **only when the daemon starts**, so after editing it run:

```bash
pkill hypr-alttab
```

The next Alt+Tab starts a fresh daemon with the new settings.

Launch parameters:
```text
  -dev
    	enable developer mode (reads the config from the dev config dir)
```

### Configuration: `~/.config/hypr-dock/hypr-alttab.conf`

```ini
[General]
# Switcher size, as a percentage of the screen (1-100). It is always centered.
WidthPercent = 100
HeightPercent = 60

# Font size (px)
FontSize = 20

# Maximum preview card width (px). Cards shrink automatically so that all
# of them fit inside the switcher; this is only the upper bound.
PreviewWidth = 400

# Space between the switcher edge and the cards (px)
Padding = 40

# Gap between cards, horizontal and vertical (px)
Spacing = 30

# false = only show workspaces on the focused monitor
ShowAllMonitors = false

# App icon badge size on cards (0 = auto, >0 = fixed px)
IconSize = 0

# true  = cycle between workspaces (each card is a whole workspace)
# false = cycle between the windows on the current workspace
CycleWorkspaces = true

# Allow the mouse to hover/click entries.
# false = keyboard only; a cursor resting over a card cannot hijack the selection
MouseSelection = false
```

#### Card sizing
Up to 4 cards are placed per row. The switcher computes how many rows it needs and picks the largest card width (up to `PreviewWidth`) that fits the switcher area after `Padding` and `Spacing`. With very many windows cards stop shrinking at 120 px and the switcher becomes scrollable.
