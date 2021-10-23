function startAnimating(canvas, spritesheetUrl, frames) {
  let stop = false;
  let start = null;
  let curr = 0;
  let image = null;

  // Set the canvas width and height, which must be the *image's* width and
  // height for the scale to be correct. This is separate from the *displayed*
  // height, which you can change by setting the *style* width and height:
  //
  // canvas.style.width = `${width}px`;
  // canvas.style.height = `${height}px`;
  //
  const [, , x1, y1, x2, y2] = frames[0][1][0];
  canvas.width = x2 - x1;
  canvas.height = y2 - y1;

  const draw = (time) => {
    if (!start) start = time;
    if (stop) return;

    const ctx = canvas.getContext("2d");
    for (; curr < frames.length; curr++) {
      if (time - start < frames[curr][0] - frames[0][0]) {
        break;
      }
      frames[curr][1].forEach(([x, y, x1, y1, x2, y2]) => {
        ctx.drawImage(image, x1, y1, x2 - x1, y2 - y1, x, y, x2 - x1, y2 - y1);
      });
    }

    if (curr === frames.length) {
      // start over
      start = null;
      curr = 0;
    }

    requestAnimationFrame(draw);
  };

  image = new Image();
  image.onload = () => requestAnimationFrame(draw);
  image.src = spritesheetUrl;

  return () => {
    stop = true;
  };
}

// Usage:
//
// const stopAnimating = startAnimating(canvas, spritesheetUrl, frameData);
// stopAnimating(); // clean up animation, e.g. when component unmounts
