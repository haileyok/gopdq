#!/usr/bin/env python3

import os
import sys
import urllib.request
from pathlib import Path
import time

TESTDATA_DIR = "testdata/images"


def download_images(num_images=50):
    """Download test images of various sizes"""

    Path(TESTDATA_DIR).mkdir(parents=True, exist_ok=True)

    print(f"Downloading {num_images} images from Lorem Picsum...")
    print()

    for i in range(1, num_images + 1):
        if i % 3 == 0:
            width, height = 400, 300
        elif i % 3 == 1:
            width, height = 800, 600
        else:
            width, height = 1920, 1080

        url = f"https://picsum.photos/{width}/{height}?random={i}"
        output_path = os.path.join(TESTDATA_DIR, f"test_image_{i}.jpg")

        try:
            print(f"Downloading image {i}/{num_images} ({width}x{height})...", end=" ")
            urllib.request.urlretrieve(url, output_path)
            time.sleep(0.1)

        except Exception as e:
            print(f"Failed to download image: {e}")

    print()
    print("Setup complete!")
    print(f"Downloaded images to: {TESTDATA_DIR}")


if __name__ == "__main__":
    num_images = int(sys.argv[1]) if len(sys.argv) > 1 else 50
    download_images(num_images)
