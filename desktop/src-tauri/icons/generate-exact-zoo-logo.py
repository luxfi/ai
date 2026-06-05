#!/usr/bin/env python3
"""
Generate the exact Zoo logo matching the PNG reference.
Looking at the actual logo, it's 3 circles with specific positioning.
"""

import subprocess
import os

def generate_exact_zoo_svg():
    """Generate the exact Zoo logo SVG."""

    # Based on visual analysis of the PNG:
    # - Green circle at top
    # - Red circle at bottom-left
    # - Blue circle at bottom-right
    # - Small yellow and cyan intersections
    # - White center
    # - NO visible magenta (or very minimal)

    svg = '''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Exact match to PNG -->
  <rect width="1024" height="1024" fill="#000000"/>

  <defs>
    <!-- Define clip paths for proper intersection handling -->
    <clipPath id="clipGreen">
      <circle cx="512" cy="340" r="190"/>
    </clipPath>

    <clipPath id="clipRed">
      <circle cx="375" cy="515" r="190"/>
    </clipPath>

    <clipPath id="clipBlue">
      <circle cx="649" cy="515" r="190"/>
    </clipPath>

    <!-- Intersection clip paths -->
    <clipPath id="clipGreenRed">
      <rect x="0" y="0" width="1024" height="1024"/>
      <circle cx="512" cy="340" r="190"/>
      <circle cx="375" cy="515" r="190"/>
    </clipPath>

    <clipPath id="clipGreenBlue">
      <rect x="0" y="0" width="1024" height="1024"/>
      <circle cx="512" cy="340" r="190"/>
      <circle cx="649" cy="515" r="190"/>
    </clipPath>
  </defs>

  <!-- Layer 1: Base circles -->

  <!-- Green circle -->
  <circle cx="512" cy="340" r="190" fill="#00A652"/>

  <!-- Red circle -->
  <circle cx="375" cy="515" r="190" fill="#ED1C24"/>

  <!-- Blue circle -->
  <circle cx="649" cy="515" r="190" fill="#2E3192"/>

  <!-- Layer 2: Two-way intersections (these override base colors) -->

  <!-- Yellow: Green ∩ Red -->
  <g clip-path="url(#clipGreen)">
    <circle cx="375" cy="515" r="190" fill="#FFF200" opacity="1"/>
  </g>

  <!-- Cyan: Green ∩ Blue -->
  <g clip-path="url(#clipGreen)">
    <circle cx="649" cy="515" r="190" fill="#00AEEF" opacity="1"/>
  </g>

  <!-- Layer 3: White center (all three overlap) -->
  <circle cx="512" cy="475" r="38" fill="#FFFFFF"/>

</svg>'''

    return svg

def generate_mathematically_perfect():
    """Generate using exact mathematical calculations."""

    # Circle parameters
    radius = 190

    # Centers forming an inverted triangle
    # Green at top, red and blue at bottom
    cx_green = 512
    cy_green = 340

    cx_red = 375
    cy_red = 515

    cx_blue = 649
    cy_blue = 515

    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Mathematical precision -->
  <rect width="1024" height="1024" fill="#000000"/>

  <!-- Define blend modes for proper color mixing -->
  <defs>
    <filter id="screen">
      <feBlend mode="screen"/>
    </filter>
  </defs>

  <!-- Three main circles -->
  <g>
    <!-- Green (top) -->
    <circle cx="{cx_green}" cy="{cy_green}" r="{radius}" fill="#00A652"/>

    <!-- Red (bottom-left) -->
    <circle cx="{cx_red}" cy="{cy_red}" r="{radius}" fill="#ED1C24"/>

    <!-- Blue (bottom-right) -->
    <circle cx="{cx_blue}" cy="{cy_blue}" r="{radius}" fill="#2E3192"/>
  </g>

  <!-- Intersections drawn as separate shapes -->

  <!-- Yellow (Green + Red) -->
  <ellipse cx="443" cy="427" rx="35" ry="50"
           transform="rotate(-35 443 427)"
           fill="#FFF200"/>

  <!-- Cyan (Green + Blue) -->
  <ellipse cx="581" cy="427" rx="35" ry="50"
           transform="rotate(35 581 427)"
           fill="#00AEEF"/>

  <!-- White center -->
  <circle cx="512" cy="475" r="37" fill="#FFFFFF"/>

</svg>'''

    return svg

def main():
    print("🎨 Generating exact Zoo logo...")

    # Generate both versions
    svg1 = generate_exact_zoo_svg()
    svg2 = generate_mathematically_perfect()

    # Save the main version
    with open('zoo-logo-final.svg', 'w') as f:
        f.write(svg2)
    print("✅ Saved zoo-logo-final.svg")

    # Generate PNGs
    print("\n📐 Generating PNG icons...")
    sizes = [16, 32, 64, 128, 256, 512, 1024]

    for size in sizes:
        cmd = ['rsvg-convert', '-w', str(size), '-h', str(size),
               'zoo-logo-final.svg', '-o', f'icon-{size}.png']
        subprocess.run(cmd, check=True)
        print(f"  ✅ icon-{size}.png")

        if size <= 512:
            cmd = ['rsvg-convert', '-w', str(size*2), '-h', str(size*2),
                   'zoo-logo-final.svg', '-o', f'icon-{size}@2x.png']
            subprocess.run(cmd, check=True)
            print(f"  ✅ icon-{size}@2x.png")

    # Create ICNS
    print("\n🍎 Creating macOS ICNS...")
    os.makedirs('icon.iconset', exist_ok=True)

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

    subprocess.run(['iconutil', '-c', 'icns', 'icon.iconset', '-o', 'icon.icns'])
    print("✅ Generated icon.icns")

    # Copy main files
    subprocess.run(['cp', 'icon-1024.png', 'icon.png'])
    subprocess.run(['cp', 'icon-1024.png', 'app-icon.png'])

    # Cleanup
    subprocess.run(['rm', '-rf', 'icon.iconset'])
    subprocess.run('rm -f icon-[0-9]*.png', shell=True)

    print("\n✨ Zoo logo generated successfully!")

if __name__ == '__main__':
    main()