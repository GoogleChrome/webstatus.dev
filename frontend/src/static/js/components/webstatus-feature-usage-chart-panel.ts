import {customElement, property} from 'lit/decorators.js';
import {WebstatusLineChartPanel} from './webstatus-line-chart-panel.js';
import {Task} from '@lit/task';
import {TemplateResult, html, nothing} from 'lit';
import {
  BROWSER_ID_TO_COLOR,
  BrowsersParameter,
  ChromiumUsageStat,
} from '../api/client.js';

@customElement('webstatus-feature-usage-chart-panel')
export class WebstatusFeatureWPTProgressChartPanel extends WebstatusLineChartPanel {
  @property({type: String})
  featureId!: string;
  createLoadingTask(): Task {
    return new Task(this, {
      args: () =>
        [this.startDate, this.endDate, this.featureId] as [Date, Date, string],
      task: async ([startDate, endDate, featureId]: [Date, Date, string]) => {
        if (
          featureId === undefined ||
          startDate === undefined ||
          endDate === undefined
        )
          return;
        await this._fetchAndAggregateData<ChromiumUsageStat>([
          {
            label: 'Chrome',
            fetchFunction: () =>
              this.apiClient.getChromiumDailyUsageStats(
                featureId,
                startDate,
                endDate,
              ),
            timestampExtractor: (dataPoint: ChromiumUsageStat): Date =>
              new Date(dataPoint.timestamp),
            valueExtractor: (dataPoint: ChromiumUsageStat): number =>
              dataPoint.usage || 0,
            tooltipExtractor: (dataPoint: ChromiumUsageStat): string =>
              `${dataPoint.usage ? dataPoint.usage * 100 : 0}%`,
          },
        ]);
      },
    });
  }
  getPanelID(): string {
    return 'feature-usage';
  }
  getPanelText(): string {
    return 'Feature Usage';
  }
  renderControls(): TemplateResult {
    return html`${nothing}`;
  }
  readonly featureSupportBrowsers: BrowsersParameter[] = ['chrome'];
  getDisplayDataChartOptionsInput(): {
    seriesColors: string[];
    vAxisTitle: string;
  } {
    // Compute seriesColors from selected browsers and BROWSER_ID_TO_COLOR
    const seriesColors = this.featureSupportBrowsers.map(browser => {
      const browserKey = browser as keyof typeof BROWSER_ID_TO_COLOR;
      return BROWSER_ID_TO_COLOR[browserKey];
    });

    return {
      seriesColors: seriesColors,
      vAxisTitle: 'Usage (%)',
    };
  }
}
