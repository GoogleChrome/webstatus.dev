#!/usr/bin/env bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

echo "========================================================"
echo "Pushing 5 Stacked Branches to origin..."
echo "========================================================"

git push --force-with-lease origin \
  feat/all-saved-search-categorizer-interface \
  feat/all-saved-search-rss-visitor-impl \
  feat/all-saved-search-email-visitor-impl \
  feat/all-saved-search-slack-visitor-impl \
  feat/all-saved-search-visitor-cutover

echo ""
echo "========================================================"
echo "Creating Pull Requests on GitHub..."
echo "========================================================"

echo "Creating PR 3A (Core Categorizer Interface)..."
gh pr create \
  --head feat/all-saved-search-categorizer-interface \
  --base main \
  --title "feat(workertypes): centralize summary categorization and export EventSummary fields (#2622)" \
  --body "- Create CategorizedSummaryVisitor interface and BaseSummaryVisitor engine in lib/workertypes to filter triggers and categorize highlights in one shared place.
- Add EventSummary.Categorize(triggers) and EventSummary.Accept(visitor, triggers) so delivery channels do not duplicate filter loops.
- Export Highlights, QueryErrors, and ResolvedQueryErrors fields directly on EventSummary struct to remove custom JSON marshaling boilerplate.
- Add unit tests for category routing, short-circuit aborts, and DTO serialization.

CONV=f70d8c59-6f49-4a69-bb74-5643e908f5b1
TAG=agy"

echo "Creating PR 3B (RSS Visitor Migration with Rich Sections)..."
gh pr create \
  --head feat/all-saved-search-rss-visitor-impl \
  --base feat/all-saved-search-categorizer-interface \
  --title "feat(backend): migrate RSS visitor to CategorizedSummaryVisitor with rich sections (#2622)" \
  --body "- Update rssVisitor to implement CategorizedSummaryVisitor.
- Render distinct sections in RSS feeds for Features Changed, Features Moved/Renamed, Features Split, and Features Deleted (removing generic Other Updates).
- Skip rendering empty feeds when HasContent() is false.
- Add table-driven tests for category and error routing across RSS feeds.

CONV=f70d8c59-6f49-4a69-bb74-5643e908f5b1
TAG=agy"

echo "Creating PR 3C (Email Digest Visitor Migration)..."
gh pr create \
  --head feat/all-saved-search-email-visitor-impl \
  --base feat/all-saved-search-rss-visitor-impl \
  --title "feat(email): migrate email digest renderer to CategorizedSummaryVisitor (#2622)" \
  --body "- Update email digest builder (templateDataGenerator) to implement CategorizedSummaryVisitor.
- Use summary.Accept(generator, triggers) to populate template data.
- Update golden tests verifying digest generation across error banners and categorized highlights.

CONV=f70d8c59-6f49-4a69-bb74-5643e908f5b1
TAG=agy"

echo "Creating PR 3D (Slack Webhook Visitor Migration)..."
gh pr create \
  --head feat/all-saved-search-slack-visitor-impl \
  --base feat/all-saved-search-email-visitor-impl \
  --title "feat(webhook): migrate Slack webhook builder to CategorizedSummaryVisitor (#2622)" \
  --body "- Update Slack payload builder (slackPayloadBuilder) to implement CategorizedSummaryVisitor.
- Add missing section handlers for VisitRemovedFeatures and VisitDeletedFeatures in Slack messages.
- Connect summary.Accept(builder, triggers) and update golden payload testdata.

CONV=f70d8c59-6f49-4a69-bb74-5643e908f5b1
TAG=agy"

echo "Creating PR 3E (Push Delivery Cutover)..."
gh pr create \
  --head feat/all-saved-search-visitor-cutover \
  --base feat/all-saved-search-slack-visitor-impl \
  --title "feat(push_delivery): delegate push notification checks to summary categorizer (#2622)" \
  --body "- Update shouldNotifyV1 inside push delivery dispatcher to call summary.Categorize(triggers).
- Propagate invalid summary parsing/categorization errors directly without silent suppression.
- Update dispatcher unit tests for email and webhook delivery evaluation.

CONV=f70d8c59-6f49-4a69-bb74-5643e908f5b1
TAG=agy"

echo ""
echo "========================================================"
echo "All 5 Stacked PRs Created Successfully!"
echo "========================================================"
