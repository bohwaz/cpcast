# cpcast

## Overview/Rationale

I'm building an [IDE for Go](https://codeperfect95.com) and needed to record
some screencasts. Unfortunately, I couldn't find any good ways to do this.
Gifs have terrible quality. Videos are too big.

Then I found a cool project called
[anim\_encoder](https://github.com/sublimehq/anim_encoder) written by Sublime
Text's creator for his website. It:

 * Captures a bunch of frames
 * Calculates the differing regions between frames
 * Saves each differing region as a new image
 * Combines all the images into a spritesheet
 * Outputs a list of frames, where each frame is a list of "draw this sprite at
   x, y" commands

Then he uses JavaScript to "render" the video/animation in realtime. This is
how the screencast on Sublime Text's website is done. If you inspect it you'll
see that it's just a canvas.

This technique is really good for screencasts, where you want high quality, but
not much changes between frames, and your FPS doesn't need to be that high.

Unfortunately, modern software sucks, package managers are dumb, and I couldn't
get scipy to install on my M1 Macbook on Big Sur to use his script. So I
decided to just rewrite the project in Go.

This program relies almost solely on the standard library and is <400loc.

## Usage

Currently only works on macOS as I'm using `screencapture` to take screenshots.

 * Install [GetWindowID](https://github.com/smokris/GetWindowID).

 * Clone this repo and run `go mod tidy`.

 * Open the app you want to record. Use `GetWindowID` to get the window ID of your app:

   ```
   GetWindowID <app name> --list
   ```
   (for example, to record TextEdit.app, `<app name>` would be `TextEdit`)

 * Take note of your app's window ID, then run:

   ```
   go run main.go -windowid=<your window id> -delay=250
   ```
   `-delay` is the time between frames (ms). If you want 10 fps, change it to
   `-delay=100`, etc.

 * Go over to your app, do what you want to do. When you're done, come back to
   your terminal, press Enter.

 * This outputs: `output/spritesheet.png`, a giant spritesheet, and
   `output/data.json`, an array of frames:

    ```
    [
      {
        "timestamp": 0,
        "changes": [
          {
            "x": 0,
            "y": 0,
            "x1": 0,
            "y1": 0,
            "x2": 0,
            "y2": 0
          }
        ]
      }
    ]
    ```

    Each frame contains a timestamp and a list of changes. For each frame, at
    `timestamp`, you should iterate through `changes`, grab the sprite at
    (x1, y1, x2, y2) in the spritesheet, and draw it in your canvas at (x, y).

    You can use canvas, WebGL, plain DOM nodes, whatever you want. I'll provide
    JavaScript samples at some point.
