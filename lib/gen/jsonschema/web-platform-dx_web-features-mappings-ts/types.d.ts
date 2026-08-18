export interface WebFeaturesMapping {
    "chrome-use-counters"?: UseCounterValue;
    "developer-signals"?:   DeveloperSignal;
    interop?:               [InteropRecord, ...InteropRecord[]];
    "mdn-docs"?:            [MdnLink, ...MdnLink[]];
    "standards-positions"?: [StandardsPositionRecord, ...StandardsPositionRecord[]];
    "state-of-surveys"?:    [SurveyRecord, ...SurveyRecord[]];
    wpt?:                   WptRecord;
}

/**
 * The usage metrics for a single web feature from chrome-use-counters.json.
 */
export interface UseCounterValue {
    percentageOfPageLoad: number;
    url:                  string;
}

/**
 * The developer signals for a single web feature from developer-signals.json.
 */
export interface DeveloperSignal {
    url:   string;
    votes: number;
}

/**
 * An array of Interop records from interop.json.
 */
export interface InteropRecord {
    label: string;
    url:   string;
    year:  number;
}

/**
 * An array of MDN documentation links from mdn-docs.json.
 */
export interface MdnLink {
    anchor: null | string;
    slug:   string;
    title:  string;
    url:    string;
}

/**
 * An array of standards position records from standards-positions.json.
 */
export interface StandardsPositionRecord {
    concerns: Concern[];
    position: Position;
    url:      string;
    vendor:   Vendor;
}

export type Concern = "API design" | "accessibility" | "annoyance" | "compatibility" | "complexity" | "dependencies" | "device independence" | "duplication" | "integration" | "internationalization" | "interoperability" | "maintenance" | "performance" | "portability" | "power" | "privacy" | "security" | "use cases" | "venue";

export type Position = "" | "blocked" | "defer" | "negative" | "neutral" | "oppose" | "positive" | "support";

export type Vendor = "mozilla" | "apple";

/**
 * An array of survey records from state-of-surveys.json.
 */
export interface SurveyRecord {
    name:        string;
    path:        string;
    question:    string;
    subQuestion: string;
    url:         string;
}

/**
 * A record of Web Platform Tests from wpt.json.
 */
export interface WptRecord {
    url: string;
}
