// Copyright 2026 Google LLC
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

// nolint:lll  // WONTFIX - for readability
package digest

const styleSnippets = `{{- define "font_family_main" -}}font-family: SF Pro, system-ui, sans-serif;{{- end -}}
{{- define "font_family_monospace" -}}font-family: Menlo, monospace;{{- end -}}
{{- define "font_weight_bold" -}}font-weight: 700;{{- end -}}
{{- define "font_weight_normal" -}}font-weight: 400;{{- end -}}
{{- define "color_text_dark" -}}color: #18181B;{{- end -}}
{{- define "color_text_medium" -}}color: #52525B;{{- end -}}
{{- define "color_bg_success" -}}background: #E6F4EA;{{- end -}}
{{- define "color_bg_info" -}}background: #E8F0FE;{{- end -}}
{{- define "color_bg_neutral" -}}background: #E4E4E7;{{- end -}}
{{- define "color_bg_light_neutral" -}}background: #F4F4F5;{{- end -}}`

const layoutStyles = `{{- define "style_body_wrapper" -}}max-width: 600px; margin: 0 auto; padding: 20px;{{- end -}}
{{- define "style_subject_header" -}}{{- template "style_text_normal" -}}; margin-top: 0; margin-bottom: 0; padding-bottom: 8px; font-weight: 700; font-size: 18px;{{- end -}}
{{- define "style_query_text" -}}{{- template "style_text_normal" -}}; margin-top: 0; margin-bottom: 0; align-self: stretch;{{- end -}}
{{- define "style_intro_wrapper" -}}width: auto; padding: 16px; background: #F4F4F5; border-radius: 4px; display: block; margin-bottom: 4px;{{- end -}}
{{- define "style_section_wrapper" -}}width: 100%; padding-top: 8px; padding-bottom: 8px; display: block;{{- end -}}
{{- define "style_card_body" -}}width: auto; padding-top: 12px; padding-bottom: 15px; padding-left: 15px; padding-right: 15px; overflow: hidden; border-bottom-right-radius: 4px; border-bottom-left-radius: 4px; border-left: 1px solid #E4E4E7; border-right: 1px solid #E4E4E7; border-bottom: 1px solid #E4E4E7; display: block; background: #FFFFFF; margin-top: 0;{{- end -}}
{{- define "style_button_link" -}}display: inline-block; padding: 10px 20px; background: #18181B; color: white; text-decoration: none; border-radius: 4px; {{- template "font_family_main" -}}; font-weight: 500; font-size: 14px;{{- end -}}`

const composedTextStyles = `{{- define "style_text_badge_title" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_bold" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_badge_description" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_normal" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 21px; word-wrap: break-word;{{- end -}}
{{- define "style_text_body" -}}{{- template "color_text_dark" -}}; font-size: 16px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 30.40px; word-wrap: break-word;{{- end -}}
{{- define "style_text_body_subtle" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 26.60px; word-wrap: break-word;{{- end -}}
{{- define "style_text_banner_bold" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_bold" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_banner_normal" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_feature_link" -}}{{- template "color_text_dark" -}}; font-size: 16px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; text-decoration: underline; line-height: 30.40px; word-wrap: break-word;{{- end -}}
{{- define "style_text_doc_link" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; text-decoration: underline; line-height: 26.60px;{{- end -}}
{{- define "style_text_doc_punctuation" -}}{{- template "color_text_medium" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; line-height: 26.60px;{{- end -}}
{{- define "style_text_date" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_monospace" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word; display: inline-block; margin-left: 10px;{{- end -}}
{{- define "style_text_browser_item" -}}{{- template "color_text_dark" -}}; font-size: 14px; {{- template "font_family_main" -}}; {{- template "font_weight_normal" -}}; word-wrap: break-word; display: inline-block; vertical-align: middle;{{- end -}}
{{- define "style_text_warning" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_main" -}}; font-style: italic; margin-top: 4px; display: block;{{- end -}}
{{- define "style_text_warning_inline" -}}{{- template "color_text_medium" -}}; font-size: 12px; {{- template "font_family_main" -}}; font-style: italic;{{- end -}}
{{- define "style_text_footer" -}}{{- template "color_text_medium" -}}; font-size: 11px; {{- template "font_weight_normal" -}}; word-wrap: break-word;{{- end -}}
{{- define "style_text_footer_link" -}}{{- template "color_text_medium" -}}; font-size: 11px; {{- template "font_weight_normal" -}}; text-decoration: underline; word-wrap: break-word;{{- end -}}`

// Replacements for Flexbox structures:
// - banner_wrapper: block (container), icon/text as inline-block
// - badge_wrapper: block (container), inner as block
// - feature_title_row: block (container), items as inline-block
// - footer: block
// These ensure better compatibility across email clients.
const EmailStyles = styleSnippets + layoutStyles + composedTextStyles + `{{- define "style_body" -}}{{- template "font_family_main" -}}; line-height: 1.5; color: #333; margin: 0; padding: 0;{{- end -}}
{{- define "style_badge_wrapper" -}}width: auto; padding-top: 12px; padding-bottom: 11px; padding-left: 15px; padding-right: 16px; overflow: hidden; border-top-left-radius: 4px; border-top-right-radius: 4px; display: block; margin-bottom: 0;{{- end -}}
{{- define "style_badge_inner_wrapper" -}}display: block;{{- end -}}
{{- define "style_change_detail_wrapper" -}}width: 100%; display: block; margin-top: 4px;{{- end -}}
{{- define "style_change_detail_inner" -}}display: block;{{- end -}}
{{- define "style_change_detail_div" -}}width: 100%; display: block; margin-top: 4px;{{- end -}}
{{- define "style_change_detail_inner_div" -}}display: block;{{- end -}}
{{- define "style_banner_wrapper" -}}width: auto; height: 50px; padding-top: 12px; padding-bottom: 11px; padding-left: 15px; padding-right: 16px; overflow: hidden; border-top-left-radius: 4px; border-top-right-radius: 4px; display: block; margin-bottom: 0;{{- end -}}
{{- define "style_banner_icon_wrapper_28" -}}height: 28px; width: 28px; display: inline-block; vertical-align: middle; margin-right: 8px;{{- end -}}
{{- define "style_banner_icon_wrapper_20" -}}height: 20px; width: 20px; display: inline-block; vertical-align: middle; margin-right: 4px;{{- end -}}
{{- define "style_img_responsive" -}}display: block; width: auto;{{- end -}}
{{- define "style_banner_text_wrapper" -}}display: inline-block; vertical-align: middle;{{- end -}}
{{- define "style_banner_browser_logos_wrapper" -}}display: inline-block; vertical-align: middle; margin-right: 8px;{{- end -}}
{{- define "style_browser_item_row" -}}width: 100%; display: block;{{- end -}}
{{- define "style_browser_item_logo_wrapper" -}}display: inline-block; vertical-align: middle; margin-right: 10px;{{- end -}}
{{- define "style_browser_item_feature_link_wrapper" -}}width: 100%; display: block; margin-top: 8px;{{- end -}}
{{- define "style_button_wrapper" -}}margin: 20px 0; text-align: center;{{- end -}}
{{- define "style_footer_wrapper" -}}width: 100%; padding-top: 16px; display: block; {{- template "font_family_main" -}};{{- end -}}
{{- define "style_footer_hr" -}}width: 100%; height: 1px; background: #E4E4E7; margin-bottom: 12px;{{- end -}}
{{- define "style_footer_text_wrapper" -}}display: block;{{- end -}}
{{- define "style_feature_title_row_wrapper" -}}width: 100%; display: block;{{- end -}}
{{- define "style_feature_title_row_inner" -}}display: inline-block; vertical-align: top;{{- end -}}
{{- define "style_split_into" -}}display: block;{{- end -}}`
