import { esbuildPlugin } from '@web/dev-server-esbuild';

const filteredLogs = ['Running in dev mode', 'Lit is in dev mode'];

export default /** @type {import("@web/test-runner").TestRunnerConfig} */ ({
  concurrency: 10,
  /** Resolve bare module imports */
  nodeResolve: {
    exportConditions: ['browser', 'development'],
  },
  // in a monorepo you need to set set the root dir to resolve modules
  rootDir: '../../',
  plugins: [esbuildPlugin({ ts: true })],
  files: [
    // Have to compile tests
    // Taken from https://github.com/open-wc/create/blob/master/src/generators/testing-wtr-ts/templates/static/web-test-runner.config.mjs
    'build/**/test/*.test.js',
  ],
  /** Filter out lit dev mode logs */
  filterBrowserLogs(log) {
    for (const arg of log.args) {
      if (typeof arg === 'string' && filteredLogs.some(l => arg.includes(l))) {
        return false;
      }
    }
    return true;
  },
});