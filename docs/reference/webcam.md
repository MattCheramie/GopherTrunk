---
slug: webcam
title: Webcam
entry_type: hardware
category: hw-personal-computers
description: A webcam is a small camera peripheral that captures video and stills for a computer, used for video calls, streaming, and recording, connected over USB or built into a laptop or monitor.
keywords: webcam, web camera, video conferencing, USB camera, streaming camera, image sensor, resolution, frame rate
infobox:
  - { label: Type, value: Input peripheral (camera) }
  - { label: Captures, value: Video and stills }
  - { label: Key specs, value: Resolution, frame rate, FOV }
  - { label: Connects via, value: USB; or built-in }
see_also: [peripheral, computer-monitor, laptop, usb, sensor]
cite_urls:
  - https://en.wikipedia.org/wiki/Webcam
---

A **webcam** is a small camera [peripheral](/reference/peripheral/) that captures video and still images for a computer, used for video calls, streaming, and recording.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="How a webcam works: light from the scene passes through a lens onto an image sensor, a small processor encodes the picture into a compressed video stream, and that stream travels over USB into the computer as a camera source that applications can read." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <path d="M30 50 L30 110 M40 46 L40 114 M50 42 L50 118"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1" stroke-dasharray="3 3">
    <path d="M52 54 L96 76 M52 80 L96 80 M52 106 L96 84"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <path d="M100 60 Q118 80 100 100 Q126 92 126 80 Q126 68 100 60 Z" fill="currentColor" fill-opacity="0.12"/>
    <rect x="140" y="60" width="34" height="40" rx="2" fill="currentColor" fill-opacity="0.1"/>
    <rect x="196" y="60" width="46" height="40" rx="2" fill="currentColor" fill-opacity="0.06"/>
    <path d="M242 80 H300"/>
    <path d="M292 74 L302 80 L292 86"/>
    <rect x="312" y="52" width="120" height="56" rx="4"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="40" y="34">scene</text>
    <text x="113" y="120">lens</text>
    <text x="157" y="114">sensor</text>
    <text x="219" y="114">encode</text>
    <text x="272" y="72">USB</text>
    <text x="372" y="78" font-size="8">computer</text>
    <text x="372" y="92" font-size="7" fill-opacity="0.85">camera source</text>
  </g>
</svg>
<figcaption>A webcam turns a scene into a data stream: light passes through a lens onto an image sensor, a small processor encodes the frames, and the compressed video travels over USB into the computer as a camera source applications can tap.</figcaption>
</figure>

## Overview

A webcam pairs an image [sensor](/reference/sensor/) and a lens with a small processor that encodes the picture into a video stream the computer can read. It is an input device: the [operating system](/reference/operating-system/) sees it as a camera source that applications tap into.

The specs that matter are *resolution* (1080p and 4K are common), *frame rate* (frames per second), and *field of view* (how wide a scene it captures). Webcams connect over [USB](/reference/usb/) or come built into a [laptop](/reference/laptop/) bezel or a [monitor](/reference/computer-monitor/).

## Key specs

A handful of numbers describe most of what a webcam can do:

| Spec | What it sets | Common values |
|------|--------------|---------------|
| Resolution | Image detail | 720p, 1080p, 4K |
| Frame rate | Motion smoothness | 30 or 60 fps |
| Field of view | How wide it sees | 65°–90°+ |
| Connection | How it attaches | USB, or built-in |

## Where it fits

The webcam became a fixture of remote work and online classes, and a step up from a built-in camera noticeably improves call quality. Beyond conferencing, the same hardware feeds streaming, security monitoring, and computer-vision projects. As a plain USB video source it slots neatly into homebrew automation — much as a software-defined radio is a USB input feeding a stream into the computer, a webcam feeds a frame stream that software can capture and process.

## Sources

[^wiki]: [Webcam](https://en.wikipedia.org/wiki/Webcam) — Wikipedia, on webcams as video-capture peripherals.
