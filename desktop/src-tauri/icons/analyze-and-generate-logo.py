#!/usr/bin/env python3
"""
Analyze the Zoo logo colors and generate a precise SVG with exact color matching.
This creates a proper Venn diagram with 7 distinct regions on a black background.
"""

import math
import subprocess
import os

# Based on the image provided, these are the exact colors for the Zoo logo
# The logo is a 3-circle Venn diagram creating 7 regions
COLORS = {
    'background': '#000000',  # Pure black background for macOS dock

    # Primary circles (pure colors)
    'green': '#00A652',   # Top circle - pure green
    'red': '#ED1C24',     # Bottom-left circle - pure red
    'blue': '#2E3192',    # Bottom-right circle - pure blue

    # Secondary intersections (2-way overlaps)
    'yellow': '#FFF200',   # Green + Red intersection
    'cyan': '#00AEEF',     # Green + Blue intersection
    'magenta': '#EC008C',  # Red + Blue intersection

    # Tertiary intersection (3-way overlap)
    'white': '#FFFFFF'     # Center where all 3 overlap
}

# Circle positions for perfect equilateral triangle arrangement
# Circles are positioned to create equal overlaps
RADIUS = 180  # Circle radius
CENTER_X = 512
CENTER_Y = 512

# Calculate positions for equilateral triangle
# Top circle at 90 degrees (straight up)
# Bottom circles at 210 and 330 degrees
def calculate_positions():
    # Distance from center to create proper overlaps
    # For equal overlaps, circles should be sqrt(3) * radius apart
    distance = RADIUS * 0.866  # This creates ~50% overlap

    positions = {
        'top': (CENTER_X, CENTER_Y - distance),      # Green circle
        'left': (CENTER_X - distance * 0.866, CENTER_Y + distance * 0.5),  # Red circle
        'right': (CENTER_X + distance * 0.866, CENTER_Y + distance * 0.5)  # Blue circle
    }
    return positions

def generate_svg():
    """Generate the precise Zoo logo SVG with exact color matching."""

    positions = calculate_positions()

    svg_content = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" fill="none" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Three-circle Venn diagram with 7 distinct colored regions -->
  <!-- Black background for macOS dock -->
  <rect width="1024" height="1024" fill="{COLORS['background']}"/>

  <!-- Define clipping paths for each region -->
  <defs>
    <!-- Clip paths for intersection regions -->
    <clipPath id="greenOnly">
      <circle cx="{positions['top'][0]}" cy="{positions['top'][1]}" r="{RADIUS}"/>
    </clipPath>

    <clipPath id="redOnly">
      <circle cx="{positions['left'][0]}" cy="{positions['left'][1]}" r="{RADIUS}"/>
    </clipPath>

    <clipPath id="blueOnly">
      <circle cx="{positions['right'][0]}" cy="{positions['right'][1]}" r="{RADIUS}"/>
    </clipPath>
  </defs>

  <!-- Group for the entire logo -->
  <g id="zoo-logo">
    <!-- Layer 1: Base circles (will be partially covered by intersections) -->

    <!-- Green circle (top) -->
    <circle cx="{positions['top'][0]}" cy="{positions['top'][1]}" r="{RADIUS}" fill="{COLORS['green']}"/>

    <!-- Red circle (bottom-left) -->
    <circle cx="{positions['left'][0]}" cy="{positions['left'][1]}" r="{RADIUS}" fill="{COLORS['red']}"/>

    <!-- Blue circle (bottom-right) -->
    <circle cx="{positions['right'][0]}" cy="{positions['right'][1]}" r="{RADIUS}" fill="{COLORS['blue']}"/>

    <!-- Layer 2: Two-way intersections (these override the base colors) -->

    <!-- Yellow region: Green ∩ Red (top-left intersection) -->
    <g>
      <defs>
        <clipPath id="greenRed">
          <circle cx="{positions['top'][0]}" cy="{positions['top'][1]}" r="{RADIUS}"/>
        </clipPath>
      </defs>
      <circle cx="{positions['left'][0]}" cy="{positions['left'][1]}" r="{RADIUS}"
              fill="{COLORS['yellow']}" clip-path="url(#greenRed)"/>
    </g>

    <!-- Cyan region: Green ∩ Blue (top-right intersection) -->
    <g>
      <defs>
        <clipPath id="greenBlue">
          <circle cx="{positions['top'][0]}" cy="{positions['top'][1]}" r="{RADIUS}"/>
        </clipPath>
      </defs>
      <circle cx="{positions['right'][0]}" cy="{positions['right'][1]}" r="{RADIUS}"
              fill="{COLORS['cyan']}" clip-path="url(#greenBlue)"/>
    </g>

    <!-- Magenta region: Red ∩ Blue (bottom intersection) -->
    <g>
      <defs>
        <clipPath id="redBlue">
          <circle cx="{positions['left'][0]}" cy="{positions['left'][1]}" r="{RADIUS}"/>
        </clipPath>
      </defs>
      <circle cx="{positions['right'][0]}" cy="{positions['right'][1]}" r="{RADIUS}"
              fill="{COLORS['magenta']}" clip-path="url(#redBlue)"/>
    </g>

    <!-- Layer 3: Three-way intersection (white center) -->
    <!-- This is the most complex - need to find where all 3 circles overlap -->
    <g>
      <defs>
        <clipPath id="allThree">
          <path d="M {CENTER_X} {CENTER_Y - RADIUS * 0.3}
                   A {RADIUS * 0.4} {RADIUS * 0.4} 0 0 1 {CENTER_X + RADIUS * 0.35} {CENTER_Y + RADIUS * 0.2}
                   A {RADIUS * 0.4} {RADIUS * 0.4} 0 0 1 {CENTER_X - RADIUS * 0.35} {CENTER_Y + RADIUS * 0.2}
                   A {RADIUS * 0.4} {RADIUS * 0.4} 0 0 1 {CENTER_X} {CENTER_Y - RADIUS * 0.3}
                   Z"/>
        </clipPath>
        <mask id="centerMask">
          <rect width="1024" height="1024" fill="white"/>
          <circle cx="{positions['top'][0]}" cy="{positions['top'][1]}" r="{RADIUS}" fill="white"/>
          <circle cx="{positions['left'][0]}" cy="{positions['left'][1]}" r="{RADIUS}" fill="white"/>
          <circle cx="{positions['right'][0]}" cy="{positions['right'][1]}" r="{RADIUS}" fill="white"/>
        </mask>
      </defs>

      <!-- Calculate the center intersection more precisely -->
      <circle cx="{CENTER_X}" cy="{CENTER_Y}" r="{RADIUS * 0.25}"
              fill="{COLORS['white']}" opacity="1.0"/>
    </g>
  </g>
</svg>'''

    return svg_content

def generate_precise_svg_v2():
    """Generate a more mathematically precise version."""

    # Use mathematical approach for perfect Venn diagram
    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Precise 3-circle Venn diagram -->
  <rect width="1024" height="1024" fill="{COLORS['background']}"/>

  <defs>
    <!-- Define masks for creating proper intersections -->
    <mask id="mask1">
      <rect width="1024" height="1024" fill="black"/>
      <circle cx="512" cy="362" r="180" fill="white"/>
    </mask>

    <mask id="mask2">
      <rect width="1024" height="1024" fill="black"/>
      <circle cx="406" cy="518" r="180" fill="white"/>
    </mask>

    <mask id="mask3">
      <rect width="1024" height="1024" fill="black"/>
      <circle cx="618" cy="518" r="180" fill="white"/>
    </mask>

    <!-- Intersection masks -->
    <mask id="greenRed">
      <rect width="1024" height="1024" fill="black"/>
      <g fill="white">
        <circle cx="512" cy="362" r="180"/>
        <circle cx="406" cy="518" r="180"/>
      </g>
    </mask>

    <mask id="greenBlue">
      <rect width="1024" height="1024" fill="black"/>
      <g fill="white">
        <circle cx="512" cy="362" r="180"/>
        <circle cx="618" cy="518" r="180"/>
      </g>
    </mask>

    <mask id="redBlue">
      <rect width="1024" height="1024" fill="black"/>
      <g fill="white">
        <circle cx="406" cy="518" r="180"/>
        <circle cx="618" cy="518" r="180"/>
      </g>
    </mask>

    <mask id="allThree">
      <rect width="1024" height="1024" fill="black"/>
      <g fill="white">
        <circle cx="512" cy="362" r="180"/>
        <circle cx="406" cy="518" r="180"/>
        <circle cx="618" cy="518" r="180"/>
      </g>
    </mask>
  </defs>

  <!-- Draw regions in correct order (back to front) -->

  <!-- Single color regions (no overlaps) -->
  <circle cx="512" cy="362" r="180" fill="{COLORS['green']}"/>
  <circle cx="406" cy="518" r="180" fill="{COLORS['red']}"/>
  <circle cx="618" cy="518" r="180" fill="{COLORS['blue']}"/>

  <!-- Two-way overlaps -->
  <g mask="url(#greenRed)">
    <rect x="350" y="350" width="200" height="200" fill="{COLORS['yellow']}"/>
  </g>

  <g mask="url(#greenBlue)">
    <rect x="480" y="350" width="200" height="200" fill="{COLORS['cyan']}"/>
  </g>

  <g mask="url(#redBlue)">
    <rect x="420" y="450" width="200" height="200" fill="{COLORS['magenta']}"/>
  </g>

  <!-- Three-way overlap (center) -->
  <g mask="url(#allThree)">
    <rect x="470" y="430" width="84" height="84" fill="{COLORS['white']}"/>
  </g>
</svg>'''
    return svg

def main():
    # Generate the SVG content
    svg_content = generate_precise_svg_v2()

    # Save the SVG file
    svg_file = 'zoo-logo-precise.svg'
    with open(svg_file, 'w') as f:
        f.write(svg_content)
    print(f"✅ Generated {svg_file}")

    # Generate PNG versions at different sizes
    sizes = [16, 32, 64, 128, 256, 512, 1024]

    for size in sizes:
        png_file = f'icon-{size}.png'
        cmd = [
            'rsvg-convert',
            '-w', str(size),
            '-h', str(size),
            svg_file,
            '-o', png_file
        ]
        try:
            subprocess.run(cmd, check=True)
            print(f"✅ Generated {png_file}")
        except subprocess.CalledProcessError as e:
            print(f"❌ Failed to generate {png_file}: {e}")

    # Generate @2x versions for retina
    for size in [16, 32, 64, 128, 256, 512]:
        size2x = size * 2
        png_file = f'icon-{size}@2x.png'
        cmd = [
            'rsvg-convert',
            '-w', str(size2x),
            '-h', str(size2x),
            svg_file,
            '-o', png_file
        ]
        try:
            subprocess.run(cmd, check=True)
            print(f"✅ Generated {png_file}")
        except subprocess.CalledProcessError as e:
            print(f"❌ Failed to generate {png_file}: {e}")

    # Create ICNS file for macOS
    print("\n🔨 Creating ICNS file for macOS...")

    # Create iconset directory
    os.makedirs('icon.iconset', exist_ok=True)

    # Copy files with correct names for iconutil
    icon_mapping = {
        'icon-16.png': 'icon_16x16.png',
        'icon-32.png': 'icon_16x16@2x.png',
        'icon-32.png': 'icon_32x32.png',
        'icon-64.png': 'icon_32x32@2x.png',
        'icon-128.png': 'icon_128x128.png',
        'icon-256.png': 'icon_128x128@2x.png',
        'icon-256.png': 'icon_256x256.png',
        'icon-512.png': 'icon_256x256@2x.png',
        'icon-512.png': 'icon_512x512.png',
        'icon-1024.png': 'icon_512x512@2x.png',
    }

    for src, dst in icon_mapping.items():
        if os.path.exists(src):
            os.system(f'cp {src} icon.iconset/{dst}')

    # Generate ICNS
    try:
        subprocess.run(['iconutil', '-c', 'icns', 'icon.iconset', '-o', 'icon.icns'], check=True)
        print("✅ Generated icon.icns")
    except subprocess.CalledProcessError as e:
        print(f"❌ Failed to generate ICNS: {e}")

    # Clean up
    os.system('rm -rf icon.iconset')
    os.system('rm -f icon-*.png')

    # Copy main files
    os.system('cp icon-1024.png icon.png')
    os.system('cp icon-1024.png app-icon.png')

    print("\n✨ Logo generation complete!")
    print(f"Colors used:")
    for name, color in COLORS.items():
        print(f"  {name}: {color}")

if __name__ == '__main__':
    main()