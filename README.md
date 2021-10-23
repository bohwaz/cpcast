# cpcast

## Overview/Rationale

I needed to record screencasts for [CodePerfect](https://codeperfect95.com) and
found that Sublime Text does them in a cool way.  The problem with screencasts
is that gifs have bad quality and videos are too big. Sublime's
[solution](https://github.com/sublimehq/anim_encoder) is to:

 * capture a bunch of frames
 * find the differing regions between frames
 * save each differing region as a new image
 * combine all the images into a spritesheet
 * output a list of frames, where each frame is a list of "draw this sprite at
   (x, y) at this time" commands

Then they render the video/animation in realtime. If you inspect the screencast
on their website, you'll see it's just a canvas being drawn to. This technique
is great for programming screencasts, where you want high visual quality, but
not much changes between frames, and your FPS can be relatively low.

Unfortunately, I couldn't get scipy to install on my M1 Macbook to use his
script -- not making this up, by the way -- so I decided to just rewrite the
project in Go. This program relies almost solely on the standard library and is
~<400loc~ <500loc.

## Examples

 * [View CodePerfect screencasts](https://codeperfect95.com)
 * [Sublime Text](https://sublimetext.com) uses the same technique, though of
   course not my library.

## Usage

Currently only macOS supported as I'm taking screenshots using `screencapture`.

 * Clone this repo and install deps with `go mod tidy`.

 * Use [`GetWindowID`](https://github.com/smokris/GetWindowID) to get the
   window ID of your app.

 * Start recording:

   ```
   go run main.go -windowid=<your window id> -delay=100 -output=./output
   ```

   Fill in your window ID. Replace `-delay` with the time between frames (ms).

 * When you're done recording, come back to your terminal and press Enter.

### Output

cpcast outputs two files: `output/spritesheet.png`, a giant spritesheet, and
`output/data.json`, an array of frames. The frames look like this:

```
[
  [
    1634692991553,
    [
      [365, 319, 0, 1252, 35, 1302],
      [90, 454, 35, 1252, 69, 1301],
      [1481, 1145, 69, 1252, 117, 1293],
      [1512, 1145, 117, 1252, 163, 1293],
      ...
    ]
  ],
  ...
]
```

 * Each frame is a [timestamp, array of changes].

 * Each change is [x, y, x1, y1, x2, y2].

 * For each change, you should draw the sprite located at (x1, y1, x2, y2)
  in the spritesheet in your canvas at (x, y).

It's up to you to render the animation using this information.  You can use
canvas, WebGL, plain DOM nodes, whatever. CodePerfect uses a [canvas](canvas.js).

The spritesheet is not optimized in any way; compresspng.com gave me about 50%
compression.

Custom logic is best handled in your renderer, like speeding up the video, or
starting a few seconds into the video, e.g. to skip the part where you focus
your app window. These things should be pretty trivial to implement in a
handrolled renderer.
