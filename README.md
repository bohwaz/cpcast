# cpcast

## Overview/Rationale

I needed to record screencasts for [CodePerfect](https://codeperfect95.com) and
found that Sublime Text's website does screencasts in a cool way. The problem
with screencasts is, gifs have bad quality and videos are too big. Sublime's
[solution](https://github.com/sublimehq/anim_encoder) is to:

 * capture a bunch of frames
 * find the differing regions between frames
 * save each differing region as a new image
 * combine all the images into a spritesheet
 * output a list of frames, where each frame is a list of "draw this sprite at
   (x, y) at this time" commands

Then they render the video/animation in realtime. If you inspect the screencast
on their website you'll see it's just a canvas being drawn to.

This technique is great for programming screencasts, where you want high visual
quality, but not much changes between frames, and your FPS doesn't need to be
that high.

I couldn't get scipy to install on my M1 Macbook to use his script (not making
this up, by the way), so I decided to just rewrite the project in Go. This
program relies almost solely on the standard library and is <400loc.

## Examples

You can see this in action [here](https://codeperfect95.com) (the screencasts
on the homepage).

The Sublime Text [website](https://sublimetext.com) uses the same technique,
though of course not my library.

## Usage

Currently only works on macOS as I'm using `screencapture` to take screenshots.

 * Clone this repo and run `go mod tidy`.

 * Use [`GetWindowID`](https://github.com/smokris/GetWindowID) to get the
   window ID of your app.

 * Run this command to start recording:

   ```
   go run main.go -windowid=<your window id> -delay=250 -output=./output
   ```

   Fill in your window ID. Replace `-delay` is the time between frames (ms). If
   you want 10 fps, change it to `-delay=100`, etc.

 * Go over to your app, do what you want to do. When you're done, come back to
   your terminal and press Enter.

 * This creates `output/spritesheet.png`, a giant spritesheet, and
   `output/data.json`, an array of frames:

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

 *  It's up to you to render the animation using this information.  You can use
    canvas, WebGL, plain DOM nodes, whatever you want.  CodePerfect uses a
    canvas. I'll provide JavaScript samples at some point.

The spritesheet is not optimized in any way; compresspng.com gave me about 50%
compression.

Custom logic is best handled in your renderer, like speeding up the video, or
starting a few seconds into the video, e.g. to skip the part where you focus
your app window. These things should be pretty trivial to implement in a
handrolled renderer.
