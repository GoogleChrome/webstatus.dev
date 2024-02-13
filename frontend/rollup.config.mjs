/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import path from 'path'
import { nodeResolve } from '@rollup/plugin-node-resolve'
// import sourcemaps from 'rollup-plugin-sourcemaps'
import terser from '@rollup/plugin-terser'
import copy from 'rollup-plugin-copy'
import css from 'rollup-plugin-css-only'
// import typescript from '@rollup/plugin-typescript'

export default [
  {
    input: 'dist/static/public/build/static/js/index.js',
    plugins: [
      // Entry point for application build; can specify a glob to build multiple
      // HTML files for non-SPA app
      // html({
      //   input: 'index.html',
      // }),
      // Resolve bare module specifiers to relative paths
      nodeResolve(),
      // Minify JS
      // sourcemaps(),
      terser({
        ecma: 2020,
        module: true,
        warnings: true
      }),
      css({
        output: 'css/styles.css'
      }),
      // typescript({
      //   sourceMap: true
      //   // inlineSources: true
      // }),
      copy({
        targets: [
          // Copy all files in img recursively.
          // Currently copying svg files from https://github.com/mdn/yari/tree/main/client/src/assets/icons/baseline
          { src: 'src/static/img/**', dest: 'dist/static/public/img' },
          // Copy the img files
          // Currently copying img files from ./scripts/postinstall.js
          { src: '.postinstall/static/img/*', dest: 'dist/static/public/img' },
          // Copy the html file
          { src: 'src/static/index.html', dest: 'dist/static' }
          // Copy sourcemaps from build
          // {
          //   src: 'dist/static/public/build/**/*.map',
          //   dest: 'dist/static/public/build'
          // },
          // // Copy source .ts files
          // { src: 'src/static/**/*.ts', dest: 'dist/static/public/ts' }
        ],
        verbose: true
      })
    ],
    output: {
      dir: 'dist/static/public',
      entryFileNames: 'js/[name].js',
      sourcemap: 'inline'
    },
    onwarn: (warning) => {
      if (warning.code === 'THIS_IS_UNDECLARED') {
        return
      }
      if (warning.code === 'THIS_IS_UNDEFINED') {
        return
      }

      console.warn(warning.message)
    },
    preserveEntrySignatures: 'strict'
  }
]
