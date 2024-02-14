export const DRAWER_WIDTH_PX = 390 // TODO: Should be whatever screenwidth is.

// Determine if the browser looks like the user is on a mobile device.
// We assume that a small enough window width implies a mobile device.
export const NARROW_WINDOW_MAX_WIDTH = 700

export const IS_MOBILE = (() => {
  // If innerWidth is non-zero, use it.
  // Otherwise, use the documentElement.clientWidth, if non-zero.
  // Otherwise, use the body.clientWidth.

  const width =
    window.innerWidth !== 0
      ? window.innerWidth
      : document.documentElement?.clientWidth !== 0
        ? document.documentElement.clientWidth
        : document.body.clientWidth

  return width <= NARROW_WINDOW_MAX_WIDTH || width === 0
})()
