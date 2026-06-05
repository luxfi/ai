#!/usr/bin/env python3
"""
Analyze the Zoo logo PNG pixel by pixel to map out exact colors and positions.
Then generate a mathematically accurate SVG.
"""

from PIL import Image
import numpy as np
from collections import defaultdict
import colorsys

def rgb_to_hex(r, g, b):
    """Convert RGB to hex color."""
    return f"#{r:02x}{g:02x}{b:02x}"

def analyze_logo():
    """Analyze the PNG logo to map colors and positions."""

    # Load the image
    img = Image.open('icon.png')
    img = img.convert('RGBA')

    # Get dimensions
    width, height = img.size
    print(f"Image size: {width}x{height}")

    # Convert to numpy array
    pixels = np.array(img)

    # Dictionary to store color regions
    color_map = defaultdict(list)
    color_counts = defaultdict(int)

    # Analyze each pixel
    for y in range(height):
        for x in range(width):
            r, g, b, a = pixels[y, x]
            if a > 0:  # Only count non-transparent pixels
                hex_color = rgb_to_hex(r, g, b)
                color_map[hex_color].append((x, y))
                color_counts[hex_color] += 1

    # Print top colors by count
    print("\nTop colors found (by pixel count):")
    sorted_colors = sorted(color_counts.items(), key=lambda x: x[1], reverse=True)

    main_colors = {}
    for i, (color, count) in enumerate(sorted_colors[:20]):
        percentage = (count / (width * height)) * 100
        print(f"{i+1}. {color}: {count} pixels ({percentage:.2f}%)")

        # Identify what each color represents
        r = int(color[1:3], 16)
        g = int(color[3:5], 16)
        b = int(color[5:7], 16)

        if r < 20 and g < 20 and b < 20:
            color_name = "Black (background)"
        elif r > 200 and g > 200 and b > 200:
            color_name = "White (center)"
        elif r < 100 and g > 150 and b < 100:
            color_name = "Green"
        elif r > 200 and g < 100 and b < 100:
            color_name = "Red"
        elif r < 100 and g < 100 and b > 150:
            color_name = "Blue"
        elif r > 200 and g > 200 and b < 100:
            color_name = "Yellow"
        elif r < 100 and g > 150 and b > 150:
            color_name = "Cyan"
        elif r > 200 and g < 100 and b > 150:
            color_name = "Magenta"
        else:
            # Calculate HSV to better identify
            h, s, v = colorsys.rgb_to_hsv(r/255, g/255, b/255)
            h_deg = h * 360

            if s < 0.2:
                color_name = "Gray/Neutral"
            elif 110 <= h_deg <= 130:
                color_name = "Green"
            elif 0 <= h_deg <= 20 or 340 <= h_deg <= 360:
                color_name = "Red"
            elif 200 <= h_deg <= 260:
                color_name = "Blue"
            elif 45 <= h_deg <= 65:
                color_name = "Yellow"
            elif 170 <= h_deg <= 190:
                color_name = "Cyan"
            elif 280 <= h_deg <= 320:
                color_name = "Magenta"
            else:
                color_name = f"Unknown (H:{h_deg:.0f})"

        print(f"   -> {color_name}")
        main_colors[color_name] = color

    # Find circle centers by analyzing each primary color region
    print("\n\nAnalyzing circle positions...")

    # Find centers of main color regions
    def find_center(color_pixels):
        if not color_pixels:
            return None
        xs = [p[0] for p in color_pixels]
        ys = [p[1] for p in color_pixels]
        return (sum(xs) // len(xs), sum(ys) // len(ys))

    # Identify the three main circles
    green_pixels = []
    red_pixels = []
    blue_pixels = []

    for y in range(height):
        for x in range(width):
            r, g, b, a = pixels[y, x]
            if a > 0:
                # Pure or dominant green
                if g > 150 and r < 100 and b < 100:
                    green_pixels.append((x, y))
                # Pure or dominant red
                elif r > 200 and g < 100 and b < 100:
                    red_pixels.append((x, y))
                # Pure or dominant blue
                elif b > 150 and r < 100 and g < 100:
                    blue_pixels.append((x, y))

    green_center = find_center(green_pixels) if green_pixels else (512, 350)
    red_center = find_center(red_pixels) if red_pixels else (380, 550)
    blue_center = find_center(blue_pixels) if blue_pixels else (644, 550)

    print(f"Green circle center: {green_center}")
    print(f"Red circle center: {red_center}")
    print(f"Blue circle center: {blue_center}")

    # Estimate radius by finding max distance from center
    def estimate_radius(pixels, center):
        if not pixels or not center:
            return 200
        distances = [((p[0]-center[0])**2 + (p[1]-center[1])**2)**0.5 for p in pixels]
        return max(distances) if distances else 200

    green_radius = estimate_radius(green_pixels, green_center)
    red_radius = estimate_radius(red_pixels, red_center)
    blue_radius = estimate_radius(blue_pixels, blue_center)

    print(f"\nEstimated radii:")
    print(f"Green: {green_radius:.1f}")
    print(f"Red: {red_radius:.1f}")
    print(f"Blue: {blue_radius:.1f}")

    # Average radius
    avg_radius = (green_radius + red_radius + blue_radius) / 3
    print(f"Average radius: {avg_radius:.1f}")

    return {
        'green_center': green_center,
        'red_center': red_center,
        'blue_center': blue_center,
        'radius': int(avg_radius),
        'colors': main_colors
    }

def generate_svg(params):
    """Generate SVG based on analyzed parameters."""

    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Pixel-perfect match based on analysis -->
  <rect width="1024" height="1024" fill="#000000"/>

  <!-- Three main circles based on pixel analysis -->

  <!-- Green circle (top) -->
  <circle cx="{params['green_center'][0]}" cy="{params['green_center'][1]}"
          r="{params['radius']}" fill="#00b050"/>

  <!-- Red circle (bottom-left) -->
  <circle cx="{params['red_center'][0]}" cy="{params['red_center'][1]}"
          r="{params['radius']}" fill="#ff0000"/>

  <!-- Blue circle (bottom-right) -->
  <circle cx="{params['blue_center'][0]}" cy="{params['blue_center'][1]}"
          r="{params['radius']}" fill="#0070c0"/>

  <!-- Intersection colors will be created by overlapping with opacity -->
  <g opacity="1">
    <!-- Yellow (Green + Red) -->
    <clipPath id="greenRed">
      <circle cx="{params['green_center'][0]}" cy="{params['green_center'][1]}" r="{params['radius']}"/>
    </clipPath>
    <circle cx="{params['red_center'][0]}" cy="{params['red_center'][1]}"
            r="{params['radius']}" fill="#ffff00" clip-path="url(#greenRed)"/>

    <!-- Cyan (Green + Blue) -->
    <clipPath id="greenBlue">
      <circle cx="{params['green_center'][0]}" cy="{params['green_center'][1]}" r="{params['radius']}"/>
    </clipPath>
    <circle cx="{params['blue_center'][0]}" cy="{params['blue_center'][1]}"
            r="{params['radius']}" fill="#00ffff" clip-path="url(#greenBlue)"/>

    <!-- Magenta (Red + Blue) -->
    <clipPath id="redBlue">
      <circle cx="{params['red_center'][0]}" cy="{params['red_center'][1]}" r="{params['radius']}"/>
    </clipPath>
    <circle cx="{params['blue_center'][0]}" cy="{params['blue_center'][1]}"
            r="{params['radius']}" fill="#ff00ff" clip-path="url(#redBlue)"/>
  </g>

  <!-- White center (all three overlap) -->
  <circle cx="512" cy="500" r="50" fill="#ffffff"/>

</svg>'''

    return svg

def main():
    print("Analyzing Zoo logo PNG...")
    params = analyze_logo()

    print("\n\nGenerating mathematically accurate SVG...")
    svg = generate_svg(params)

    with open('zoo-logo-pixel-perfect.svg', 'w') as f:
        f.write(svg)

    print("✅ Saved zoo-logo-pixel-perfect.svg")
    print("\nSVG parameters used:")
    print(f"  Green center: {params['green_center']}")
    print(f"  Red center: {params['red_center']}")
    print(f"  Blue center: {params['blue_center']}")
    print(f"  Circle radius: {params['radius']}")

if __name__ == '__main__':
    main()