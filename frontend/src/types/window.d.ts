declare global {
  interface Window extends Window {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    dataLayer: Record<any, any>[];
    /**
     * The `gtag` function for Google Analytics.
     *
     * @param {string} command - The command for gtag.
     * @param {any} param - Required parameter.
     * @param {Object} [params] - Additional parameters.
     */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    gtag: (command: string, param: any, params?: any) => void;
  }
}

// empty export to keep file a module
export {};
