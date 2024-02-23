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
  .hbox.reverse {
    flex-direction: row-reverse;
  }
  .vbox.reverse {
    flex-direction: column-reverse;
  }

  .hbox.inline,
  .vbox.inline {
    display: inline-flex;
  }

  .hbox.wrap,
  .vbox.wrap {
    flex-wrap: wrap;
  }
  .hbox.wrap-reverse,
  .vbox.wrap-reverse {
    flex-wrap: wrap-reverse;
  }

  /* Flexbox alignment */

  /* Alignment in main axis. */
  .hbox.halign-items-start,
  .vbox.valign-items-start {
    justify-content: flex-start;
  }
  .hbox.valign-items-start,
  .vbox.halign-items-start {
    align-content: flex-start;
  }

  .hbox.halign-items-center,
  .vbox.valign-items-center {
    justify-content: center;
  }

  .hbox.halign-items-end,
  .vbox.valign-items-end {
    justify-content: flex-end;
  }

  /* Alignment in cross axis. */
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

  /* Space distribution */

  .hbox.halign-items-space-around,
  .vbox.valign-items-space-around {
    justify-content: space-around;
  }
  .hbox.valign-items-space-around,
  .vbox.halign-items-space-around {
    align-content: space-around;
  }

  /* obsolete - use .halign-space-between */
  .hbox.space-between {
    justify-content: space-between;
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

  /* Self/item alignment. */
  .hbox > .valign-start,
  .vbox > .halign-start {
    align-self: flex-start;
  }

  .hbox > .valign-center,
  .vbox > .halign-center {
    align-self: center;
  }

  .hbox > .valign-end,
  .vbox > .halign-end {
    align-self: flex-end;
  }

  /* Strech "alignment" */
  .hbox.valign-items-stretch,
  .vbox.halign-items-stretch {
    align-items: baseline; /* ??? */
  }

  /* obsolete - use .halign-stretch */
  .hbox > .hgrow {
    flex-grow: 1;
  }

  .hbox > .halign-stretch {
    flex-grow: 1;
  }
  .hbox > .valign-stretch {
    align-self: stretch;
  }
  .vbox > .halign-stretch {
    align-self: stretch;
  }
  .vbox > .valign-stretch {
    flex-grow: 1;
  }

  .hbox.halign-items-stretch > *,
  .vbox.valign-items-stretch > * {
    flex-grow: 1;
  }

  /* Non-flexbox positioning helper styles */

  .hbox > .halign-stretch-1,
  .vbox > .valign-stretch-1 {
    flex-basis: 0.000000001px; /* ??? */
  }

  .hbox > .halign-stretch-2,
  .vbox > .valign-stretch-2 {
    flex-grow: 2;
  }

  .hbox > .halign-stretch-3,
  .vbox > .valign-stretch-3 {
    flex-grow: 3;
  }
`;
