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

import {assert, html} from '@open-wc/testing';
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
    // Spy on the chartWrapper.draw method (make sure chartWrapper is initialized)
    const drawSpy = sinon.spy(component.chartWrapper!, 'draw');

    // Simulate a resize
    const resizeObserverCallback = mockResizeObserver.args[0][0];
    resizeObserverCallback([
      {
        contentRect: {width: 200, height: 100},
      },
    ]);

    // Assert that chartWrapper.draw was called
    assert.isTrue(drawSpy.calledOnce);
  });
});
