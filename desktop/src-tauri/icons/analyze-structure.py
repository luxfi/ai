#!/usr/bin/env python3
from PIL import Image
import numpy as np

# Load the original logo
img = Image.open('original-zoo-logo.png')
pixels = np.array(img)
width, height = img.size

print(f'Image size: {width}x{height}')
print('\nAnalyzing structure:')

# The logo appears to be a perfect circle
# Let's understand what's inside vs outside

center_x, center_y = 100, 100
radius = 100

# Check specific points
test_points = [
    (100, 100, 'Center'),
    (100, 50, 'Top center'),
    (50, 100, 'Left center'),
    (150, 100, 'Right center'),
    (100, 150, 'Bottom center'),
    (70, 70, 'Top-left diagonal'),
    (130, 70, 'Top-right diagonal'),
    (70, 130, 'Bottom-left diagonal'),
    (130, 130, 'Bottom-right diagonal'),
    # Points that should be in gaps
    (60, 60, 'Outside top-left'),
    (140, 60, 'Outside top-right'),
    (100, 20, 'Far top'),
    (100, 180, 'Far bottom'),
]

for x, y, label in test_points:
    if 0 <= x < width and 0 <= y < height:
        pixel = pixels[y][x]
        r, g, b, a = pixel
        if a == 0:
            color = 'TRANSPARENT'
        elif r > 250 and g > 250 and b > 250:
            color = 'WHITE'
        elif r > 200 and g < 50:
            color = 'RED'
        elif g > 150 and r < 50:
            color = 'GREEN'
        elif b > 140 and r < 50:
            color = 'BLUE'
        elif r > 200 and g > 200 and b < 50:
            color = 'YELLOW'
        elif g > 150 and b > 200:
            color = 'CYAN'
        elif r > 200 and b > 140:
            color = 'MAGENTA'
        else:
            color = f'RGB({r},{g},{b})'

        print(f'{label:20} ({x:3},{y:3}): {color:15} (alpha={a})')

# The key realization: The gaps between circles INSIDE the circular boundary
# should be TRANSPARENT, not filled with any color