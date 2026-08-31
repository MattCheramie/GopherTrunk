"""Shared paths/constants for the GopherTrunk video pipeline."""
import glob
import os

import imageio_ffmpeg

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
VIDEO = os.path.join(REPO, "video")
RENDER = os.path.join(VIDEO, "_render")
FF = imageio_ffmpeg.get_ffmpeg_exe()
CHROME = sorted(glob.glob("/opt/pw-browsers/chromium-*/chrome-linux/chrome"))[-1] if glob.glob(
    "/opt/pw-browsers/chromium-*/chrome-linux/chrome") else "chromium"
CHROME_ARGS = ["--no-sandbox", "--force-color-profile=srgb", "--hide-scrollbars",
               "--allow-file-access-from-files", "--disable-lcd-text"]
FPS = 30
PIPER_MODEL = os.path.join(RENDER, "tts", "en-us-ryan-high.onnx")

os.makedirs(RENDER, exist_ok=True)
