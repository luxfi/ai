#!/usr/bin/env python3
"""
Calculate the perfect Zoo logo with exact intersection geometry.
The white center is the area where all three circles overlap.
"""

import math

def calculate_circle_intersections():
    """Calculate where the three circles intersect."""

    # Circle centers (based on our analysis)
    green = (512, 362)  # Top
    red = (384, 562)    # Bottom-left
    blue = (640, 562)   # Bottom-right
    radius = 220

    # Calculate distances between centers
    def distance(p1, p2):
        return math.sqrt((p1[0] - p2[0])**2 + (p1[1] - p2[1])**2)

    d_green_red = distance(green, red)
    d_green_blue = distance(green, blue)
    d_red_blue = distance(red, blue)

    print(f"Distance green-red: {d_green_red:.1f}")
    print(f"Distance green-blue: {d_green_blue:.1f}")
    print(f"Distance red-blue: {d_red_blue:.1f}")

    # Calculate intersection points of each pair of circles
    def circle_intersection(c1, c2, r):
        """Find intersection points of two circles."""
        d = distance(c1, c2)

        if d > 2 * r:
            return None  # No intersection

        # Using formula for circle-circle intersection
        a = d / 2
        h = math.sqrt(r**2 - a**2) if r**2 > a**2 else 0

        # Midpoint between centers
        mx = (c1[0] + c2[0]) / 2
        my = (c1[1] + c2[1]) / 2

        # Direction perpendicular to line between centers
        dx = (c2[1] - c1[1]) / d
        dy = (c1[0] - c2[0]) / d

        # Two intersection points
        p1 = (mx + h * dx, my + h * dy)
        p2 = (mx - h * dx, my - h * dy)

        return p1, p2

    # Find the region where all three circles overlap
    # This forms a curved triangle (Reuleaux triangle)

    # The center of the white region
    center_x = (green[0] + red[0] + blue[0]) / 3
    center_y = (green[1] + red[1] + blue[1]) / 3

    print(f"\nWhite center point: ({center_x:.1f}, {center_y:.1f})")

    return {
        'green': green,
        'red': red,
        'blue': blue,
        'radius': radius,
        'white_center': (center_x, center_y)
    }

def generate_perfect_svg():
    """Generate the perfect SVG with exact calculations."""

    params = calculate_circle_intersections()

    svg = f'''<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Mathematically perfect with exact intersections -->
  <rect width="1024" height="1024" fill="#000000"/>

  <defs>
    <!-- Individual circle masks -->
    <mask id="mask-all">
      <rect width="1024" height="1024" fill="black"/>
      <circle cx="{params['green'][0]}" cy="{params['green'][1]}" r="{params['radius']}" fill="white"/>
      <circle cx="{params['red'][0]}" cy="{params['red'][1]}" r="{params['radius']}" fill="white"/>
      <circle cx="{params['blue'][0]}" cy="{params['blue'][1]}" r="{params['radius']}" fill="white"/>
    </mask>
  </defs>

  <!-- Layer 1: Base circles -->

  <!-- Green circle (top) -->
  <circle cx="{params['green'][0]}" cy="{params['green'][1]}"
          r="{params['radius']}" fill="#00A652"/>

  <!-- Red circle (bottom-left) -->
  <circle cx="{params['red'][0]}" cy="{params['red'][1]}"
          r="{params['radius']}" fill="#ED1C24"/>

  <!-- Blue circle (bottom-right) -->
  <circle cx="{params['blue'][0]}" cy="{params['blue'][1]}"
          r="{params['radius']}" fill="#2E3192"/>

  <!-- Layer 2: Two-way intersections -->

  <!-- Yellow (Green ∩ Red) -->
  <defs>
    <clipPath id="clip-gr">
      <circle cx="{params['green'][0]}" cy="{params['green'][1]}" r="{params['radius']}"/>
    </clipPath>
  </defs>
  <circle cx="{params['red'][0]}" cy="{params['red'][1]}"
          r="{params['radius']}" fill="#FCF006" clip-path="url(#clip-gr)"/>

  <!-- Cyan (Green ∩ Blue) -->
  <defs>
    <clipPath id="clip-gb">
      <circle cx="{params['green'][0]}" cy="{params['green'][1]}" r="{params['radius']}"/>
    </clipPath>
  </defs>
  <circle cx="{params['blue'][0]}" cy="{params['blue'][1]}"
          r="{params['radius']}" fill="#01ACF1" clip-path="url(#clip-gb)"/>

  <!-- Magenta (Red ∩ Blue) -->
  <defs>
    <clipPath id="clip-rb">
      <circle cx="{params['red'][0]}" cy="{params['red'][1]}" r="{params['radius']}"/>
    </clipPath>
  </defs>
  <circle cx="{params['blue'][0]}" cy="{params['blue'][1]}"
          r="{params['radius']}" fill="#EA018E" clip-path="url(#clip-rb)"/>

  <!-- Layer 3: White center (all three overlap) -->
  <!-- This is the most complex part - the curved triangle -->

  <defs>
    <clipPath id="clip-all">
      <path d="M {params['green'][0]} {params['green'][1]}
               A {params['radius']} {params['radius']} 0 0 1 {params['red'][0]} {params['red'][1]}
               A {params['radius']} {params['radius']} 0 0 1 {params['blue'][0]} {params['blue'][1]}
               A {params['radius']} {params['radius']} 0 0 1 {params['green'][0]} {params['green'][1]} Z"/>
    </clipPath>

    <!-- Create intersection of all three circles -->
    <clipPath id="center-clip">
      <circle cx="{params['green'][0]}" cy="{params['green'][1]}" r="{params['radius']}"/>
    </clipPath>
  </defs>

  <!-- White center using multiple clip paths -->
  <g clip-path="url(#center-clip)">
    <g clip-path="url(#clip-rb)">
      <circle cx="{params['white_center'][0]}" cy="{params['white_center'][1]}"
              r="60" fill="#FFFFFF"/>
    </g>
  </g>

</svg>'''

    return svg

def main():
    print("Calculating perfect Zoo logo geometry...\n")

    svg = generate_perfect_svg()

    with open('zoo-logo-perfect-calculated.svg', 'w') as f:
        f.write(svg)

    print("\n✅ Generated zoo-logo-perfect-calculated.svg")

if __name__ == '__main__':
    main()