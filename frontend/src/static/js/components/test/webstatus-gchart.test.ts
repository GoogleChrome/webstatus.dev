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

import {assert, expect, html} from '@open-wc/testing';
import '../../services/webstatus-gcharts-loader-service.js';
import '../webstatus-gchart.js';
import {type WebstatusGChart} from '../webstatus-gchart.js';
import {render} from 'lit';
import {type WebstatusGChartsLoaderService} from '../../services/webstatus-gcharts-loader-service.js';
import sinon from 'sinon';

describe('webstatus-gchart', () => {
  let mockResizeObserver: sinon.SinonStub;
  let component: WebstatusGChart;
  let loaderComponent: WebstatusGChartsLoaderService;

  beforeEach(async () => {
    // Mock the ResizeObserver constructor to prevent
    // "ResizeObserver loop completed with undelivered notifications."
    // errors that can happen intermittently in tests.
    // An added benefit is that we can control the resize.
    mockResizeObserver = sinon.stub(window, 'ResizeObserver').callsFake(() => ({
      observe: sinon.stub(),
      disconnect: sinon.stub(),
    }));

    // Create a root div and append it to the body
    const root = document.createElement('div');
    document.body.appendChild(root);

    // Use render to create the components
    render(
      html`
        <webstatus-gcharts-loader-service>
          <div id="test-container">
            <webstatus-gchart
              .containerId="${'test-container'}"
            ></webstatus-gchart>
          </div>
        </webstatus-gcharts-loader-service>
      `,
      root,
    );

    // Get the components
    loaderComponent = root.querySelector(
      'webstatus-gcharts-loader-service',
    ) as WebstatusGChartsLoaderService;
    component = loaderComponent.querySelector(
      'webstatus-gchart',
    ) as WebstatusGChart;

    component.currentSelection = [{row: 0, column: 1}];

    await loaderComponent.updateComplete;
    await loaderComponent.waitForGChartsLibraryLoaded();
    await component.updateComplete;
  });

  afterEach(() => {
    sinon.restore();
  });

  it('can receive loaded state via loader context', async () => {
    assert.equal(component.gchartsLibraryLoaded, true);
  });

  it('redraws the chart on resize', async () => {
    // Spy on the chartWrapper.draw and setSelection methods (make sure chartWrapper is initialized)
    const drawSpy = sinon.spy(component.chartWrapper!, 'draw');
    const mockChart = sinon.createStubInstance(google.visualization.LineChart);
    sinon.stub(component.chartWrapper!, 'getChart').returns(mockChart);

    // Simulate a resize
    const resizeObserverCallback = mockResizeObserver.args[0][0];
    resizeObserverCallback([
      {
        contentRect: {width: 200, height: 100},
      },
    ]);

    // Assert that chartWrapper.draw was called
    assert.isTrue(drawSpy.calledOnce);
    assert.isTrue(mockChart.setSelection.calledOnceWith([{row: 0, column: 1}]));
  });

  describe('Selection events', () => {
    beforeEach(async () => {
      // Set up the chart with some data and options
      component.dataObj = {
        cols: [
          {type: 'date', label: 'Date', role: 'domain'},
          {type: 'number', label: 'Value 1', role: 'data'},
          {type: 'number', label: 'Value 2', role: 'data'},
        ],
        rows: [
          [new Date('2024-01-01'), 10, 20],
          [new Date('2024-01-02'), 20, 30],
        ],
      };
      component.options = {};
      await component.updateComplete;
    });

    it('dispatches point-selected event on data point click', async () => {
      const chart =
        component.chartWrapper!.getChart() as google.visualization.LineChart;
      const dispatchEventSpy = sinon.spy(component, 'dispatchEvent');

      // Simulate a click on the first data point of the second series ('Value 2')
      chart.setSelection([{row: 0, column: 2}]);
      google.visualization.events.trigger(chart, 'select');

      await component.updateComplete;

      expect(dispatchEventSpy).to.have.been.calledOnce;
      const dispatchedEvent = dispatchEventSpy.getCall(0)
        .args[0] as unknown as CustomEvent;
      expect(dispatchedEvent.type).to.equal('point-selected');
      expect(dispatchedEvent.detail).to.deep.equal({
        label: 'Value 2',
        timestamp: new Date('2024-01-01'),
        value: 20,
      });
    });

    it('dispatches point-deselected event on deselection', async () => {
      const chart =
        component.chartWrapper!.getChart() as google.visualization.LineChart;
      const dispatchEventSpy = sinon.spy(component, 'dispatchEvent');

      // Simulate a click to select a data point
      chart.setSelection([{row: 0, column: 1}]);
      google.visualization.events.trigger(chart, 'select');

      // Simulate a click on an empty area to deselect
      chart.setSelection([]);
      google.visualization.events.trigger(chart, 'select');

      // Assert that point-deselected was dispatched
      expect(dispatchEventSpy).to.have.been.calledTwice; // Called for select and deselect
      const selectEvent = dispatchEventSpy.getCall(0)
        .args[0] as unknown as CustomEvent;
      expect(selectEvent.type).to.equal('point-selected');
      expect(selectEvent.detail).to.deep.equal({
        label: 'Value 1',
        timestamp: new Date('2024-01-01'),
        value: 10,
      });
      const deselectEvent = dispatchEventSpy.getCall(1)
        .args[0] as unknown as CustomEvent; // Get the second call (deselect)
      expect(deselectEvent.type).to.equal('point-deselected');
      expect(deselectEvent.detail).to.be.null;
    });
  });
});
