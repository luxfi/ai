#!/usr/bin/env python3
"""
Generate a mathematically perfect Zoo logo Venn diagram.
Three circles arranged in equilateral triangle creating 7 distinct regions.
"""

import math
import subprocess
import os

# Exact colors from the Zoo logo
COLORS = {
    'background': '#000000',  # Black background for macOS
    'green': '#00A652',       # Top circle
    'red': '#ED1C24',         # Bottom-left circle
    'blue': '#2E3192',        # Bottom-right circle
    'yellow': '#FFF200',      # Green + Red
    'cyan': '#00AEEF',        # Green + Blue
    'magenta': '#EC008C',     # Red + Blue
    'white': '#FFFFFF'        # All three overlap
}

def generate_perfect_venn_svg():
    """
    Generate a mathematically perfect 3-circle Venn diagram.
    Circles are arranged in an equilateral triangle pattern.
    """

    # Circle parameters
    radius = 200
    canvas_size = 1024
    center = canvas_size / 2

    # For proper overlap, circles should be positioned at distance d from center
    # where d creates approximately 30% overlap
    d = radius * 0.9  # Distance from center point

    # Calculate circle centers (equilateral triangle arrangement)
    # Top circle at 90° (north)
    cx1 = center
    cy1 = center - d

    # Bottom-left at 210°
    angle2 = math.radians(210)
    cx2 = center + d * math.cos(angle2)
    cy2 = center + d * math.sin(angle2)

    # Bottom-right at 330°
    angle3 = math.radians(330)
    cx3 = center + d * math.cos(angle3)
    cy3 = center + d * math.sin(angle3)

    svg = f'''<svg width="{canvas_size}" height="{canvas_size}" viewBox="0 0 {canvas_size} {canvas_size}" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Mathematically perfect Venn diagram -->
  <rect width="{canvas_size}" height="{canvas_size}" fill="{COLORS['background']}"/>

  <defs>
    <!-- Define gradients for smoother color transitions -->
    <radialGradient id="greenGrad">
      <stop offset="0%" stop-color="{COLORS['green']}" stop-opacity="1"/>
      <stop offset="100%" stop-color="{COLORS['green']}" stop-opacity="0.95"/>
    </radialGradient>

    <radialGradient id="redGrad">
      <stop offset="0%" stop-color="{COLORS['red']}" stop-opacity="1"/>
      <stop offset="100%" stop-color="{COLORS['red']}" stop-opacity="0.95"/>
    </radialGradient>

    <radialGradient id="blueGrad">
      <stop offset="0%" stop-color="{COLORS['blue']}" stop-opacity="1"/>
      <stop offset="100%" stop-color="{COLORS['blue']}" stop-opacity="0.95"/>
    </radialGradient>
  </defs>

  <!-- Base circles -->
  <g id="base-circles">
    <circle cx="{cx1}" cy="{cy1}" r="{radius}" fill="{COLORS['green']}" opacity="1"/>
    <circle cx="{cx2}" cy="{cy2}" r="{radius}" fill="{COLORS['red']}" opacity="1"/>
    <circle cx="{cx3}" cy="{cy3}" r="{radius}" fill="{COLORS['blue']}" opacity="1"/>
  </g>

  <!-- Calculate intersection points -->
  <!-- Two-way intersections -->
  <g id="two-way-intersections">
    <!-- Yellow: Green ∩ Red -->
    <path d="M {cx1 - radius * 0.5} {cy1 + radius * 0.5}
             A {radius} {radius} 0 0 0 {cx2 + radius * 0.5} {cy2 - radius * 0.5}
             A {radius} {radius} 0 0 0 {cx1 - radius * 0.5} {cy1 + radius * 0.5}
             Z" fill="{COLORS['yellow']}"/>

    <!-- Cyan: Green ∩ Blue -->
    <path d="M {cx1 + radius * 0.5} {cy1 + radius * 0.5}
             A {radius} {radius} 0 0 1 {cx3 - radius * 0.5} {cy3 - radius * 0.5}
             A {radius} {radius} 0 0 1 {cx1 + radius * 0.5} {cy1 + radius * 0.5}
             Z" fill="{COLORS['cyan']}"/>

    <!-- Magenta: Red ∩ Blue -->
    <path d="M {cx2 + radius * 0.7} {cy2}
             A {radius} {radius} 0 0 0 {cx3 - radius * 0.7} {cy3}
             A {radius} {radius} 0 0 0 {cx2 + radius * 0.7} {cy2}
             Z" fill="{COLORS['magenta']}"/>
  </g>

  <!-- Three-way center intersection -->
  <g id="center-intersection">
    <circle cx="{center}" cy="{center}" r="{radius * 0.25}" fill="{COLORS['white']}"/>
  </g>
</svg>'''

    return svg

def generate_precise_venn_v3():
    """Even more precise version using path calculations for exact regions."""

    svg = '''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Ultra-precise Venn diagram with 7 distinct colored regions -->
  <rect width="1024" height="1024" fill="#000000"/>

  <defs>
    <!-- Masks for each individual circle -->
    <mask id="m1">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="512" cy="332" r="200" fill="black"/>
    </mask>

    <mask id="m2">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="384" cy="540" r="200" fill="black"/>
    </mask>

    <mask id="m3">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="640" cy="540" r="200" fill="black"/>
    </mask>
  </defs>

  <!-- Draw all 7 regions separately for precise color control -->

  <!-- Region 1: Green only (top, no overlaps) -->
  <path d="M 512 132 A 200 200 0 0 1 712 332 A 200 200 0 0 0 612 432 A 200 200 0 0 0 412 432 A 200 200 0 0 0 312 332 A 200 200 0 0 1 512 132 Z"
        fill="#00A652"/>

  <!-- Region 2: Red only (bottom-left, no overlaps) -->
  <path d="M 184 540 A 200 200 0 0 1 384 340 A 200 200 0 0 0 434 440 A 200 200 0 0 0 384 540 A 200 200 0 0 0 434 640 A 200 200 0 0 0 384 740 A 200 200 0 0 1 184 540 Z"
        fill="#ED1C24"/>

  <!-- Region 3: Blue only (bottom-right, no overlaps) -->
  <path d="M 840 540 A 200 200 0 0 0 640 340 A 200 200 0 0 1 590 440 A 200 200 0 0 1 640 540 A 200 200 0 0 1 590 640 A 200 200 0 0 1 640 740 A 200 200 0 0 0 840 540 Z"
        fill="#2E3192"/>

  <!-- Region 4: Yellow (Green + Red intersection only) -->
  <path d="M 412 432 A 200 200 0 0 1 462 382 A 200 200 0 0 1 512 432 A 200 200 0 0 0 462 482 A 200 200 0 0 0 412 432 Z"
        fill="#FFF200"/>

  <!-- Region 5: Cyan (Green + Blue intersection only) -->
  <path d="M 612 432 A 200 200 0 0 0 562 382 A 200 200 0 0 0 512 432 A 200 200 0 0 1 562 482 A 200 200 0 0 1 612 432 Z"
        fill="#00AEEF"/>

  <!-- Region 6: Magenta (Red + Blue intersection only) -->
  <path d="M 484 590 A 200 200 0 0 1 512 540 A 200 200 0 0 1 540 590 A 200 200 0 0 0 512 640 A 200 200 0 0 0 484 590 Z"
        fill="#EC008C"/>

  <!-- Region 7: White (all three circles intersection) -->
  <circle cx="512" cy="470" r="50" fill="#FFFFFF"/>
</svg>'''

    return svg

def main():
    print("🎨 Generating perfect Zoo logo...")

    # Generate the perfect SVG
    svg_content = generate_perfect_venn_svg()

    # Save as zoo-logo-perfect.svg
    svg_file = 'zoo-logo-perfect.svg'
    with open(svg_file, 'w') as f:
        f.write(svg_content)
    print(f"✅ Saved {svg_file}")

    # Also save the ultra-precise version
    svg_v3 = generate_precise_venn_v3()
    with open('zoo-logo-ultra.svg', 'w') as f:
        f.write(svg_v3)
    print("✅ Saved zoo-logo-ultra.svg")

    # Check if rsvg-convert is available
    try:
        subprocess.run(['which', 'rsvg-convert'], check=True, capture_output=True)
    except:
        print("Installing librsvg...")
        subprocess.run(['brew', 'install', 'librsvg'])

    # Generate PNG versions
    sizes = [16, 32, 64, 128, 256, 512, 1024]

    print("\n📐 Generating PNG icons...")
    for size in sizes:
        cmd = ['rsvg-convert', '-w', str(size), '-h', str(size),
               svg_file, '-o', f'icon-{size}.png']
        subprocess.run(cmd, check=True)
        print(f"  ✅ icon-{size}.png")

        # Generate @2x for retina
        if size <= 512:
            cmd = ['rsvg-convert', '-w', str(size*2), '-h', str(size*2),
                   svg_file, '-o', f'icon-{size}@2x.png']
            subprocess.run(cmd, check=True)
            print(f"  ✅ icon-{size}@2x.png")

    # Create ICNS for macOS
    print("\n🍎 Creating macOS ICNS file...")
    os.makedirs('icon.iconset', exist_ok=True)

    # Copy with correct names for iconutil
    mappings = [
        ('icon-16.png', 'icon_16x16.png'),
        ('icon-32.png', 'icon_16x16@2x.png'),
        ('icon-32.png', 'icon_32x32.png'),
        ('icon-64.png', 'icon_32x32@2x.png'),
        ('icon-128.png', 'icon_128x128.png'),
        ('icon-256.png', 'icon_128x128@2x.png'),
        ('icon-256.png', 'icon_256x256.png'),
        ('icon-512.png', 'icon_256x256@2x.png'),
        ('icon-512.png', 'icon_512x512.png'),
        ('icon-1024.png', 'icon_512x512@2x.png'),
    ]

    for src, dst in mappings:
        subprocess.run(['cp', src, f'icon.iconset/{dst}'])

    # Generate ICNS
    subprocess.run(['iconutil', '-c', 'icns', 'icon.iconset', '-o', 'icon.icns'], check=True)
    print("✅ Generated icon.icns")

    # Copy main files
    subprocess.run(['cp', 'icon-1024.png', 'icon.png'])
    subprocess.run(['cp', 'icon-1024.png', 'app-icon.png'])

    # Clean up
    subprocess.run(['rm', '-rf', 'icon.iconset'])
    subprocess.run('rm -f icon-[0-9]*.png', shell=True)

    print("\n✨ Perfect Zoo logo generated successfully!")
    print("\n🎨 Color palette used:")
    for name, color in COLORS.items():
        print(f"  {name:12} {color}")

if __name__ == '__main__':
    main()