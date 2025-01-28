#!/bin/bash
rm -rf 600
mkdir 600
go run .
cd 600
ffmpeg -framerate 60 -i ./btcusd_%03d.png -c:v libvpx-vp9 -pix_fmt yuva420p -lossless 1 out.webm
