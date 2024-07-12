#!/bin/bash

# Resize all the images in the current directory to a suitable size for the website and add a border.
#
# Requires imagemagick
#
# > sudo apt-get install imagemagick

for file in *.jpg; do convert -resize 700x700\> -bordercolor '#f5f5f5' -border 12 "$file" "$file"; done
