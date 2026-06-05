#!/bin/bash

# Generate high-resolution PNG icons from Zoo SVG logo
# Optimized for macOS Sequoia dock display

SVG_FILE="zoo-logo-final.svg"
if [ ! -f "$SVG_FILE" ]; then
    echo "Error: $SVG_FILE not found!"
    echo "Looking for alternative SVG files..."
    if [ -f "zoo-logo-accurate.svg" ]; then
        SVG_FILE="zoo-logo-accurate.svg"
        echo "Using $SVG_FILE"
    else
        echo "No suitable SVG file found!"
        exit 1
    fi
fi

# Check for required tools
if ! command -v rsvg-convert &> /dev/null; then
    echo "rsvg-convert not found. Installing via homebrew..."
    brew install librsvg
fi

if ! command -v convert &> /dev/null; then
    echo "ImageMagick convert not found. Installing via homebrew..."
    brew install imagemagick
fi

# Generate high-resolution dock icon (1024x1024 for retina displays)
echo "Generating high-resolution dock icon..."
rsvg-convert -w 1024 -h 1024 "$SVG_FILE" -o icon-1024.png

# Generate all required sizes for macOS
SIZES=(16 32 64 128 256 512 1024)
for size in "${SIZES[@]}"; do
    echo "Generating ${size}x${size} icon..."
    rsvg-convert -w $size -h $size "$SVG_FILE" -o "icon-${size}.png"
    
    # Also generate 2x versions for retina
    double=$((size * 2))
    if [ $double -le 2048 ]; then
        echo "Generating ${size}x${size}@2x icon..."
        rsvg-convert -w $double -h $double "$SVG_FILE" -o "icon-${size}@2x.png"
    fi
done

# Generate menu bar icon (must be template image style for macOS)
echo "Generating menu bar icon..."
# Menu bar icons should be 22pt (44px@2x) and use template image style
rsvg-convert -w 22 -h 22 "$SVG_FILE" -o tray-icon-macos-temp.png
# Convert to monochrome for template image
convert tray-icon-macos-temp.png -colorspace gray -alpha copy -channel RGB -fill black +opaque none tray-icon-macos.png
rm tray-icon-macos-temp.png

# Generate @2x version for retina displays
rsvg-convert -w 44 -h 44 "$SVG_FILE" -o tray-icon-macos-temp@2x.png
convert tray-icon-macos-temp@2x.png -colorspace gray -alpha copy -channel RGB -fill black +opaque none tray-icon-macos@2x.png
rm tray-icon-macos-temp@2x.png

# Copy main icons
cp icon-1024.png icon.png
cp icon-1024.png app-icon.png
cp icon-128.png 128x128.png
cp icon-256@2x.png 128x128@2x.png 2>/dev/null || cp icon-512.png 128x128@2x.png
cp icon-32.png 32x32.png

# Generate ICNS file for macOS (most important for dock display)
echo "Generating ICNS file with all required resolutions..."
mkdir -p icon.iconset

# Copy icons with proper naming for iconutil
cp icon-16.png icon.iconset/icon_16x16.png
cp icon-32.png icon.iconset/icon_16x16@2x.png
cp icon-32.png icon.iconset/icon_32x32.png
cp icon-64.png icon.iconset/icon_32x32@2x.png
cp icon-128.png icon.iconset/icon_128x128.png
cp icon-256.png icon.iconset/icon_128x128@2x.png
cp icon-256.png icon.iconset/icon_256x256.png
cp icon-512.png icon.iconset/icon_256x256@2x.png
cp icon-512.png icon.iconset/icon_512x512.png
cp icon-1024.png icon.iconset/icon_512x512@2x.png

# Create the ICNS file
iconutil -c icns icon.iconset -o icon.icns
rm -rf icon.iconset

# Generate ICO file for Windows
echo "Generating ICO file for Windows..."
convert icon-16.png icon-32.png icon-48.png icon-64.png icon-128.png icon-256.png icon.ico 2>/dev/null || \
convert icon-16.png icon-32.png icon-64.png icon-128.png icon-256.png icon.ico

# Clean up temporary files
echo "Cleaning up temporary files..."
rm -f icon-*[0-9].png icon-*[0-9]@2x.png icon-48.png 2>/dev/null

echo "Icon generation complete!"
echo "Generated:"
echo "  - icon.icns (macOS dock icon with all resolutions)"
echo "  - icon.png (main icon 1024x1024)"
echo "  - app-icon.png (application icon 1024x1024)"
echo "  - icon.ico (Windows icon)"
echo "  - tray-icon-macos.png (menu bar icon)"
echo "  - tray-icon-macos@2x.png (menu bar icon @2x)"
echo ""
echo "The ICNS file now contains all proper resolutions for macOS Sequoia dock display."