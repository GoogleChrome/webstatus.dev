// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package digest

// defaultEmailTemplate is the main layout. It uses {{template "name" .}} to include
// the components defined in emailComponents.
// nolint: lll  // WONTFIX - Keeping for readability.
const defaultEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{.Subject}}</title>
</head>
<body style='{{- template "style_body" -}}'>
    <div style='{{- template "style_body_wrapper" -}}'>
        {{- template "intro_text" . -}}

        {{- if .BaselineNewlyChanges -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "banner_baseline_newly" dict "LogoURL" (statusLogoURL "newly") -}}
            {{- range .BaselineNewlyChanges -}}
                {{- $date := "" -}}
                {{- if .BaselineChange.To.LowDate -}}
                    {{- $date = formatDate .BaselineChange.To.LowDate -}}
                {{- end -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs "Date" $date -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .BaselineWidelyChanges -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "banner_baseline_widely" dict "LogoURL" (statusLogoURL "widely") -}}
            {{- range .BaselineWidelyChanges -}}
                {{- $date := "" -}}
                {{- if .BaselineChange.To.HighDate -}}
                    {{- $date = formatDate .BaselineChange.To.HighDate -}}
                {{- end -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs "Date" $date -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .BaselineRegressionChanges -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "banner_baseline_regression" dict "LogoURL" (statusLogoURL "limited") -}}
            {{- range .BaselineRegressionChanges -}}
                {{- $date := "" -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs "Date" $date -}}
                    <div style='{{- template "style_change_detail_div" -}}'>
                        <div style='{{- template "style_change_detail_inner_div" -}}'>
                            <span style='{{- template "style_text_body_subtle" -}}'>
                                {{- with .BaselineChange -}}
                                From {{formatBaselineStatus .From.Status}}
                                {{- end -}}
                            </span>
                        </div>
                    </div>
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .AllBrowserChanges -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "banner_browser_implementation" -}}
            {{- range .AllBrowserChanges -}}
                {{- template "browser_item" dict "Name" (browserDisplayName .BrowserName) "LogoURL" (browserLogoURL .BrowserName) "From" .Change.From "To" .Change.To "FeatureName" .FeatureName "FeatureURL" (printf "%s/features/%s" $.BaseURL .FeatureID) -}}
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .AddedFeatures -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "badge" (dict "Title" "Added" "Description" "These features now match your search criteria.") -}}
            {{- range .AddedFeatures -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .RemovedFeatures -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "badge" (dict "Title" "Removed" "Description" "These features no longer match your search criteria.") -}}
            {{- range .RemovedFeatures -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .DeletedFeatures -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "badge" (dict "Title" "Deleted" "Description" "These features have been removed from the web platform.") -}}
            {{- range .DeletedFeatures -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .MovedFeatures -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "badge" (dict "Title" "Moved" "Description" "These features have been renamed or merged with another feature.") -}}
            {{- range .MovedFeatures -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs -}}
                    {{- template "change_detail" dict "Label" "Moved from" "From" .Moved.From.Name "To" .Moved.To.Name -}}
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .SplitFeatures -}}
        <div style='{{- template "style_section_wrapper" -}}'>
            {{- template "badge" (dict "Title" "Split" "Description" "This feature has been split into multiple, more granular features.") -}}
            {{- range .SplitFeatures -}}
                <div style='{{- template "style_card_body" -}}'>
                    {{- template "feature_title_row" dict "Name" .FeatureName "URL" (printf "%s/features/%s" $.BaseURL .FeatureID) "Docs" .Docs -}}
                    <div style='{{- template "style_change_detail_div" -}}'>
                        <div style='{{- template "style_split_into" -}}'>
                            <span style='{{- template "style_text_body" -}}'>Split into</span>
                            {{ range $i, $feature := .Split.To -}}
                                {{- if $i }}, {{ end -}}
                                <a href="{{printf "%s/features/%s" $.BaseURL $feature.ID}}" style='{{- template "style_text_doc_link" -}}'>{{$feature.Name}}</a>
                            {{- end -}}
                        </div>
                    </div>
                </div>
            {{- end -}}
        </div>
        {{- end -}}

        {{- if .Truncated -}}
            {{- template "button" dict "URL" (printf "%s/saved-searches" $.BaseURL) "Text" "View All Changes" -}}
        {{- end -}}

        {{- template "footer" dict "UnsubscribeURL" $.UnsubscribeURL "ManageURL" (printf "%s/saved-searches" $.BaseURL) -}}
    </div>
</body>
</html>`
