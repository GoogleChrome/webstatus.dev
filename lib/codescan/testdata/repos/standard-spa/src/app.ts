// Standard SPA application entry point

export function setupApp() {
  // TODO(baseline/popover): replace custom tooltip with native popover
  const tooltip = document.querySelector('.tooltip');

  // TODO(baseline/view-transitions): smooth page transitions
  if (document.startViewTransition) {
    document.startViewTransition();
  }
}
