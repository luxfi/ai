#!/usr/bin/env python3
import subprocess
import os
import tempfile
import shutil

def create_icns():
    """Create macOS ICNS file with all required resolutions"""

    # Create a temporary iconset directory
    with tempfile.TemporaryDirectory() as tmpdir:
        iconset_path = os.path.join(tmpdir, "icon.iconset")
        os.makedirs(iconset_path)

        # Required icon sizes for macOS
        sizes = [
            (16, "icon_16x16.png"),
            (32, "icon_16x16@2x.png"),
            (32, "icon_32x32.png"),
            (64, "icon_32x32@2x.png"),
            (128, "icon_128x128.png"),
            (256, "icon_128x128@2x.png"),
            (256, "icon_256x256.png"),
            (512, "icon_256x256@2x.png"),
            (512, "icon_512x512.png"),
            (1024, "icon_512x512@2x.png"),
        ]

        # Generate all required sizes
        for size, filename in sizes:
            output_path = os.path.join(iconset_path, filename)
            cmd = [
                "rsvg-convert",
                "-w", str(size),
                "-h", str(size),
                "zoo-logo-perfect-circle-final.svg",
                "-o", output_path
            ]
            subprocess.run(cmd, check=True)
            print(f"Generated {filename} ({size}x{size})")

        # Create the ICNS file
        subprocess.run([
            "iconutil",
            "-c", "icns",
            iconset_path,
            "-o", "icon.icns"
        ], check=True)

        print("\n✅ Successfully created icon.icns")

        # Show file size for verification
        size = os.path.getsize("icon.icns")
        print(f"📦 ICNS file size: {size:,} bytes ({size/1024:.1f} KB)")

if __name__ == "__main__":
    create_icns()