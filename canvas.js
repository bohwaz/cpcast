class Anim {
  constructor(canvas, spritesheetUrl, frames) {
    this.canvas = canvas;
    this.frames = frames;

    this.stop = false;
    this.start = null;
    this.frame = 0;

    this.image = new Image();
    this.image.src = spritesheetUrl;
    this.image.onload = () => {
      if (!this.stop) {
        requestAnimationFrame(this.draw);
      }
    };

    // Set the canvas width and height, which must be the *image's* width and
    // height for the scale to be correct. This is separate from the
    // *displayed* height, which you can change by setting the *style* width
    // and height:
    //
    // this.canvas.style.width = `${width}px`;
    // this.canvas.style.height = `${height}px`;

    const [, , x1, y1, x2, y2] = this.frames[0][1][0];
    this.canvas.width = x2 - x1;
    this.canvas.height = y2 - y1;
  }

  cleanup() {
    this.stop = true;
  }

  draw = (time) => {
    if (!this.start) this.start = time;
    if (this.stop) return;

    const ctx = this.canvas.getContext("2d");

    while (
      this.frame < this.frames.length &&
      time - this.start > this.frames[this.frame][0] - this.frames[0][0]
    ) {
      this.frames[this.frame][1].forEach((change) => {
        const [x, y, x1, y1, x2, y2] = change;
        const [w, h] = [x2 - x1, y2 - y1];
        ctx.drawImage(this.image, x1, y1, w, h, x, y, w, h);
      });
      this.frame++;
    }

    if (this.frame === this.frames.length) {
      // start over
      this.start = null;
      this.frame = 0;
    }

    requestAnimationFrame(this.draw);
  };
}
