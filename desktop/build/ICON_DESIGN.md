# ScheduleGate Icon Design

## Design Concept

The ScheduleGate icon embodies the core purpose of the tool: **gatekeeping schedule quality** through DCMA 14-Point Assessment.

## Visual Elements

### 1. **Gantt Chart Bars** (Left Side)
- Four horizontal bars representing project schedule timelines
- Gradient cyan-blue color (`#00d9ff` → `#0099ff`)
- Varying opacity creates depth and visual hierarchy
- Symbolizes: Project scheduling, timeline management, MS Project integration

### 2. **Gate/Shield Badge** (Right Side)
- Green shield/checkpoint indicator (`#00ff88` → `#00cc6a`)
- Contains a checkmark symbol
- Glowing effect for emphasis
- Symbolizes: Quality gate, compliance check, assessment approval

### 3. **Dark Background**
- Deep navy gradient (`#0f0f1a` → `#1a1a2e`)
- Modern, professional aesthetic
- Matches dark theme preference
- Provides high contrast for icon elements

### 4. **SG Monogram** (Subtle)
- "SG" text in background (15% opacity)
- Brand reinforcement without visual clutter
- Positioned at bottom center

### 5. **Frame Border**
- Rounded rectangle frame
- Cyan accent stroke
- Creates visual containment and polish

## Color Palette

| Color | Hex | Usage |
|-------|-----|-------|
| Dark Background | `#0f0f1a` → `#1a1a2e` | Primary background gradient |
| Cyan Accent | `#00d9ff` → `#0099ff` | Gantt bars, borders, highlights |
| Green Success | `#00ff88` → `#00cc6a` | Gate/checkpoint indicator |
| Black | `#0f0f1a` | Checkmark stroke |

## File Formats Generated

```
desktop/build/
├── icon.svg          # Master vector file (scalable)
├── icon.png          # 512x512 PNG
├── icon.ico          # Windows ICO (multi-size: 16-256px)
├── icon_16x16.png    # 16x16 PNG (favicon, taskbar)
├── icon_32x32.png    # 32x32 PNG
├── icon_48x48.png    # 48x48 PNG
├── icon_64x64.png    # 64x64 PNG
── icon_128x128.png  # 128x128 PNG
├── icon_256x256.png  # 256x256 PNG
├── icon_512x512.png  # 512x512 PNG
└── appicon.png       # Wails app icon (copy of 512x512)
```

## Usage

### Desktop Application (Wails)
The icon is automatically used by Wails v3 from `desktop/build/appicon.png` and `desktop/build/windows/icon.ico`.

### Web Reports
Use `icon_32x32.png` or `icon_16x16.png` as favicon in HTML reports.

### Marketing/Materials
Use `icon.svg` for print materials, presentations, and documentation.

## Design Philosophy

- **Modern & Sharp**: Clean lines, gradient fills, glowing effects
- **Dark Theme**: Professional appearance matching developer tools aesthetic
- **Symbolic**: Every element represents a core function (schedule + assessment)
- **Scalable**: Vector-based design ensures crisp rendering at any size
- **Memorable**: Unique combination of Gantt chart + gate/checkpoint imagery

## Regeneration

To regenerate icons from the SVG master:

```bash
cd desktop/build
cairosvg icon.svg -o icon.png --output-width 512 --output-height 512
python3 << 'PYEOF'
from PIL import Image
img = Image.open('icon.png')
sizes = [16, 32, 48, 64, 128, 256, 512]
for size in sizes:
    resized = img.resize((size, size), Image.Resampling.LANCZOS)
    resized.save(f'icon_{size}x{size}.png')
img.save('icon.ico', format='ICO', sizes=[(s, s) for s in [16, 32, 48, 64, 128, 256]])
PYEOF
cp icon.png appicon.png
```

## License

© 2026 ScheduleGate. All rights reserved.
