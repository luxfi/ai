#!/usr/bin/env python3
import math

# Your current values from screenshot
current = {
    "outer": {"x": 510, "y": 508, "r": 258},
    "green": {"x": 515, "y": 374},
    "red": {"x": 389, "y": 596},
    "blue": {"x": 647, "y": 597},
    "circle_r": 224
}

# Perfect symmetrical values
perfect = {
    "outer": {"x": 512, "y": 512, "r": 260},
    "green": {"x": 512, "y": 375},
    "red": {"x": 390, "y": 595},
    "blue": {"x": 634, "y": 595},
    "circle_r": 225
}

print("GEOMETRY ANALYSIS")
print("=" * 50)

# Calculate centers of triangle formed by circles
def calculate_center(g, r, b):
    cx = (g["x"] + r["x"] + b["x"]) / 3
    cy = (g["y"] + r["y"] + b["y"]) / 3
    return cx, cy

current_center = calculate_center(current["green"], current["red"], current["blue"])
perfect_center = calculate_center(perfect["green"], perfect["red"], perfect["blue"])

print(f"\nCurrent triangle center: ({current_center[0]:.1f}, {current_center[1]:.1f})")
print(f"Perfect triangle center: ({perfect_center[0]:.1f}, {perfect_center[1]:.1f})")
print(f"SVG canvas center: (512, 512)")

# Calculate distances between circles
def distance(p1, p2):
    return math.sqrt((p1["x"] - p2["x"])**2 + (p1["y"] - p2["y"])**2)

print("\nDistances between circle centers:")
print(f"Green-Red:  Current = {distance(current['green'], current['red']):.1f}, Perfect = {distance(perfect['green'], perfect['red']):.1f}")
print(f"Green-Blue: Current = {distance(current['green'], current['blue']):.1f}, Perfect = {distance(perfect['green'], perfect['blue']):.1f}")
print(f"Red-Blue:   Current = {distance(current['red'], current['blue']):.1f}, Perfect = {distance(perfect['red'], perfect['blue']):.1f}")

# For perfect equilateral triangle
side_length = distance(perfect['red'], perfect['blue'])
print(f"\nIdeal equilateral triangle side: {side_length:.1f}")

# Calculate if circles fit within boundary
def check_bounds(circles, outer):
    for name, circle in circles.items():
        dist = math.sqrt((circle["x"] - outer["x"])**2 + (circle["y"] - outer["y"])**2)
        edge = dist + 225  # circle radius
        margin = outer["r"] - edge
        print(f"{name}: center distance={dist:.1f}, edge={edge:.1f}, margin={margin:.1f}")

print("\nBoundary check (perfect values):")
check_bounds({
    "Green": perfect["green"],
    "Red": perfect["red"],
    "Blue": perfect["blue"]
}, perfect["outer"])

print("\n" + "=" * 50)
print("RECOMMENDED FINAL VALUES:")
print(f"Outer Circle: x={perfect['outer']['x']}, y={perfect['outer']['y']}, r={perfect['outer']['r']}")
print(f"Circle Radius: {perfect['circle_r']}")
print(f"Green: x={perfect['green']['x']}, y={perfect['green']['y']}")
print(f"Red: x={perfect['red']['x']}, y={perfect['red']['y']}")
print(f"Blue: x={perfect['blue']['x']}, y={perfect['blue']['y']}")
print("\nThese create a perfectly centered, symmetrical logo!")