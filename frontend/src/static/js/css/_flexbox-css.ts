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

import {css} from 'lit';

/**
 * Flexbox layout helper classes.
 *
 * This is a simplified version of the flexbox layout helper classes from
 * https://htmlpreview.github.io/?https://raw.githubusercontent.com/dlaliberte/standards-notes/master/flexbox-classes.html
 */
export const FLEX_BOX = css`
  .hbox,
  .vbox {
    display: flex;
  }

  .hbox {
    flex-direction: row;
  }
  .vbox {
    flex-direction: column;
  }

  /* Flexbox alignment */

  .hbox.halign-items-start,
  .vbox.valign-items-start {
    justify-content: flex-start;
  }
  .hbox.halign-items-center,
  .vbox.valign-items-center {
    justify-content: center;
  }
  .hbox.halign-items-end,
  .vbox.valign-items-end {
    justify-content: flex-end;
  }

  /* Alignment in cross-axis. */
  .hbox.valign-items-start,
  .vbox.halign-items-start {
    align-items: flex-start;
  }

  .hbox.valign-items-center,
  .vbox.halign-items-center {
    align-items: center;
  }

  .hbox.valign-items-end,
  .vbox.halign-items-end {
    align-items: flex-end;
  }

  /* Self/item alignment. */

  .hbox > .valign-start,
  .vbox > .halign-start,
  .hbox.valign-items-start > *,
  .vbox.halign-items-start > * {
    align-self: flex-start;
  }

  .hbox > .valign-center,
  .vbox > .halign-center,
  .hbox.valign-items-center > *,
  .vbox.halign-items-center > * {
    align-self: center;
  }

  .hbox > .valign-end,
  .vbox > .halign-end,
  .hbox.valign-items-end > *,
  .vbox.halign-items-end > * {
    align-self: flex-end;
  }

  /* Stretch "alignment" */

  .hbox > .hgrow,  /* obsolete - use .halign-stretch */
  .hbox > .halign-stretch,
  .vbox > .valign-stretch,
  .hbox.halign-items-stretch > *,
  .vbox.valign-items-stretch > * {
    flex-grow: 1;
  }
  .hbox > .halign-stretch-2,
  .vbox > .valign-stretch-2 {
    flex-grow: 2;
  }
  .hbox > .halign-stretch-3,
  .vbox > .valign-stretch-3 {
    flex-grow: 3;
  }

  .hbox.valign-items-stretch,
  .vbox.halign-items-stretch {
    align-items: stretch;
  }

  .hbox > .valign-stretch,
  .vbox > .halign-stretch,
  .hbox.valign-items-stretch > *,
  .vbox.halign-items-stretch > * {
    align-self: stretch;
  }

  /* Space distribution */

  .hbox.space-between {
    /* obsolete - use .halign-space-between */
    justify-content: space-between;
  }

  .hbox.halign-items-space-around,
  .vbox.valign-items-space-around {
    justify-content: space-around;
  }
  .hbox.valign-items-space-around,
  .vbox.halign-items-space-around {
    align-content: space-around;
  }

  .hbox.halign-items-space-between,
  .vbox.valign-items-space-between {
    justify-content: space-between;
  }
  .hbox.valign-items-space-between,
  .vbox.halign-items-space-between {
    align-content: space-between;
  }

  .hbox.halign-items-space-evenly,
  .vbox.valign-items-space-evenly {
    justify-content: space-evenly;
  }
  .hbox.valign-items-space-evenly,
  .vbox.halign-items-space-evenly {
    align-content: space-evenly;
  }

  .hbox > .spacer,
  .hbox > .spacer-1,
  .vbox > .spacer,
  .vbox > .spacer-1 {
    flex-grow: 1;
    visibility: hidden;
  }
  .hbox > .spacer-2,
  .vbox > .spacer-2 {
    flex-grow: 2;
    visibility: hidden;
  }
  .hbox > .spacer-3,
  .vbox > .spacer-3 {
    flex-grow: 3;
    visibility: hidden;
  }
`;
