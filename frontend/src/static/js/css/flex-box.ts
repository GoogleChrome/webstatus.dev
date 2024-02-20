import {css} from 'lit';

export const FLEX_BOX = css`
  .hbox {
    display: flex;
    flex-direction: row;
  }
  .hbox.space-between {
    justify-content: space-between;
  }
  .hbox > * {
    display: inline-block;
  }
  .hbox > .hgrow {
    flex-grow: 1;
  }
  .hbox > .hshrink {
    flex-shrink: 2;
  }
`;
