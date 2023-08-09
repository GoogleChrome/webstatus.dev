import { html, TemplateResult } from 'lit';
import '../src/webstatus-dash.js';

export default {
  title: 'WebstatusDash',
  component: 'webstatus-dash',
  argTypes: {
    backgroundColor: { control: 'color' },
  },
};

interface Story<T> {
  (args: T): TemplateResult;
  args?: Partial<T>;
  argTypes?: Record<string, unknown>;
}

interface ArgTypes {
  header?: string;
  backgroundColor?: string;
}

const Template: Story<ArgTypes> = ({ header, backgroundColor = 'white' }: ArgTypes) => html`
  <webstatus-dash style="--webstatus-dash-background-color: ${backgroundColor}" .header=${header}></webstatus-dash>
`;

export const App = Template.bind({});
App.args = {
  header: 'My app',
};
