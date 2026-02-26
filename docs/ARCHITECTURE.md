# Webstatus.dev Architecture & Workflows

This document contains high-level architectural diagrams for the webstatus.dev ecosystem. These diagrams are intended to help new developers understand the system boundaries, data flows, and local development environment.

_Note: The nodes in these diagrams are clickable! Click on a component to jump to its source code or relevant GCP documentation._

---

## 1. High-Level Architecture (Multi-Project Setup)

To limit the scope of the `allUsers` IAM permission (which makes the frontend and API public), the infrastructure is split across three GCP projects. They are networked together via a Shared VPC in the Host project.

```mermaid
flowchart TD
    subgraph Host_Project ["Project: Host (VPC)"]
        VPC["Shared VPC / Cloud Armor"]
    end

    subgraph Public_Project ["Project: Public (allUsers Access)"]
        FE["Frontend (Cloud Run)"]
        BE["Backend API (Cloud Run)"]
    end

    subgraph Internal_Project ["Project: Internal (Data & Jobs)"]
        direction TB
        DB[(Cloud Spanner)]
        Cache[(Valkey/Memorystore)]
        PS[[Pub/Sub]]
        Workflows["Ingestion Jobs (Cloud Run Jobs)"]
        Workers["Notification Workers"]
        GCS[(GCS State Buckets)]
    end

    %% Interaction Flows
    User((User/Browser)) --> FE
    User --> BE
    FE -.-> VPC
    BE -.-> VPC
    VPC -.-> DB
    VPC -.-> Cache
    BE --> DB
    BE --> Cache

    Workflows --> DB
    Workers --> DB
    Workers --> PS
    Workers --> GCS

    %% Clickable Links
    click VPC "https://cloud.google.com/vpc/docs/shared-vpc" "GCP Shared VPC Docs"
    click FE "https://github.com/GoogleChrome/webstatus.dev/tree/main/frontend/" "Go to Frontend Source"
    click BE "https://github.com/GoogleChrome/webstatus.dev/tree/main/backend/" "Go to Backend Source"
    click DB "https://cloud.google.com/spanner/docs" "GCP Spanner Docs"
    click Cache "https://cloud.google.com/memorystore/docs/valkey" "GCP Memorystore for Valkey"
    click PS "https://cloud.google.com/pubsub/docs" "GCP Pub/Sub Docs"
    click Workflows "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/" "Go to Workflows Source"
    click Workers "https://github.com/GoogleChrome/webstatus.dev/tree/main/workers/" "Go to Workers Source"
    click GCS "https://cloud.google.com/storage/docs" "GCP Cloud Storage Docs"
```

---

## 2. Public-to-Internal Request Flow

This diagram illustrates how a user's request travels from the browser, hits the public-facing API, and securely queries the internal database using the Spanner Adapter pattern.

```mermaid
sequenceDiagram
    participant U as User / Browser
    box "Public Project" #f9f9f9
        participant FE as Frontend (Lit)
        participant BE as Backend (Go API)
    end
    box "Internal Project" #e1f5fe
        participant C as Valkey Cache
        participant DB as Spanner DB
    end

    U->>FE: Load Dashboard
    FE->>BE: GET /v1/features
    BE->>C: Check Cache
    alt Cache Hit
        C-->>BE: Return JSON
    else Cache Miss
        BE->>DB: Query via Spanner Adapter
        DB-->>BE: Return Rows
        BE->>C: Update Cache
    end
    BE-->>FE: HTTP 200 JSON
    FE-->>U: Render Web Components
```

_(Note: Sequence diagrams in Mermaid currently have limited support for external hyperlinks on participants, so we rely on the flowchart diagrams for deep-linking.)_

---

## 3. Data Ingestion Pipeline (Internal Project)

Webstatus.dev relies heavily on external data. Cloud Scheduler triggers jobs that download, parse, and synchronize this data into our Spanner database.

```mermaid
flowchart LR
    subgraph External ["External Sources"]
        WPT["WPT.fyi API"]
        BCD["MDN BCD Repo"]
        WF["Web-Features Repo"]
    end

    subgraph Internal ["Project: Internal"]
        direction TB
        Sched["Cloud Scheduler"] --> Job["Ingestion Job (Go)"]

        subgraph Job_Logic ["Job Internal Flow"]
            DL["pkg/data/downloader.go"] --> P["pkg/data/parser.go"]
            P --> Adp["Spanner Adapter"]
        end

        Adp --> DB[(Cloud Spanner)]
    end

    WPT --> DL
    BCD --> DL
    WF --> DL

    %% Clickable Links
    click WPT "https://wpt.fyi/" "Web Platform Tests API"
    click BCD "https://github.com/mdn/browser-compat-data" "MDN BCD Repository"
    click WF "https://github.com/web-platform-dx/web-features" "Web-Features Repository"
    click Sched "https://cloud.google.com/scheduler/docs" "GCP Cloud Scheduler"
    click Job "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/" "Go to Workflow Services Source"
    click Adp "https://github.com/GoogleChrome/webstatus.dev/tree/main/lib/gcpspanner/spanneradapters/" "Go to Spanner Adapters"
    click DB "https://cloud.google.com/spanner" "GCP Spanner"
```

---

## 4. Notification System Architecture (Event-Driven)

When data changes (via ingestion workflows) or users update their saved searches, an event-driven architecture processes those changes to deliver email and push notifications.

```mermaid
flowchart TD
    subgraph Internal ["Project: Internal"]
        direction TB
        DB[(Spanner: Subscriptions & Channels)]
        EP["Event Producer Worker"] --> PS[[Pub/Sub: feature-diffs]]
        PS --> PDW["Push Delivery Worker"]
        PS --> EW["Email Worker"]

        PDW --> DB
        EW --> Temp["lib/email (Templates)"]
        EW --> Chime["Chime (email service)"]

        EP --> GCS[(GCS: Snapshot State)]
    end

    DB -- Trigger Change --> EP

    %% Clickable Links
    click DB "https://github.com/GoogleChrome/webstatus.dev/tree/main/lib/gcpspanner/" "Go to Spanner Schema/Mappers"
    click EP "https://github.com/GoogleChrome/webstatus.dev/tree/main/workers/event_producer/" "Go to Event Producer Source"
    click PS "https://cloud.google.com/pubsub" "GCP Pub/Sub"
    click PDW "https://github.com/GoogleChrome/webstatus.dev/tree/main/workers/push_delivery/" "Go to Push Delivery Worker Source"
    click EW "https://github.com/GoogleChrome/webstatus.dev/tree/main/workers/email/" "Go to Email Worker Source"
    click Temp "https://github.com/GoogleChrome/webstatus.dev/tree/main/lib/email/" "Go to HTML Email Templates"
    click GCS "https://cloud.google.com/storage" "GCS Blob Storage"
```

---

## 5. Local Development & E2E Testing Environment

Understanding the local dev loop is crucial. We use Skaffold and Minikube to orchestrate live services alongside GCP emulators. The `Makefile` provides targets to populate these emulators with either _fake data_ (for deterministic E2E testing) or _live data_ (for manual workflow testing).

```mermaid
flowchart TD
    subgraph Dev_Machine ["Developer Machine / CI Pipeline"]
        direction LR
        Make["Makefile"]
        Playwright{"Playwright (E2E Tests)"}
    end

    subgraph Minikube ["Minikube Local Cluster"]
        direction TB

        subgraph Live_Services ["Skaffold Live Services"]
            FE["Frontend (Lit/Nginx)"]
            BE["Backend API (Go)"]
        end

        subgraph Emulators ["GCP Emulators & Mocks"]
            Spanner[(Spanner Emulator)]
            DS[(Datastore Emulator)]
            Auth["Firebase Auth Emulator"]
            WM["Wiremock
(Mock GitHub API)"]
            Valkey[(Valkey Cache)]
            PS[[Pub/Sub Emulator]]
        end

        FE <-->|API Calls| BE
        FE <-->|Firebase Login| Auth
        BE <-->|Read/Write| Spanner
        BE <-->|Cache| Valkey
        BE <-->|Fetch Profile/Emails
on Login| WM
    end

    subgraph Data_Population ["Data Population Strategies"]
        direction TB
        FakeUsers["make dev_fake_users
(util/cmd/load_test_users)"]
        FakeData["make dev_fake_data
(util/cmd/load_fake_data)"]
        RealFlows["make dev_workflows
(util/run_job.sh)"]
    end

    %% Makefile connections
    Make -.->|Triggers| FakeUsers
    Make -.->|Triggers| FakeData
    Make -.->|Triggers| RealFlows

    %% Population flows
    FakeUsers -->|Seeds Test Users| Auth
    FakeData -->|Seeds Predictable Entities| Spanner
    FakeData -->|Seeds Predictable Entities| DS
    RealFlows -->|Ingests Live Data| Spanner

    %% Testing flows
    Playwright -.->|Requires| FakeData
    Playwright -.->|Requires| FakeUsers
    Playwright -->|Runs Tests Against| FE

    %% Clickable Links
    click Make "https://github.com/GoogleChrome/webstatus.dev/tree/main/Makefile" "Go to Makefile"
    click Playwright "https://github.com/GoogleChrome/webstatus.dev/tree/main/e2e/tests/" "Go to E2E Playwright Tests"
    click FE "https://github.com/GoogleChrome/webstatus.dev/tree/main/frontend/src/" "Go to Frontend Source"
    click BE "https://github.com/GoogleChrome/webstatus.dev/tree/main/backend/pkg/httpserver/" "Go to Backend Handlers"
    click Spanner "https://github.com/GoogleChrome/webstatus.dev/tree/main/.dev/spanner/" "Spanner Emulator Setup"
    click WM "https://github.com/GoogleChrome/webstatus.dev/tree/main/.dev/wiremock/" "Wiremock Configuration"
    click Auth "https://github.com/GoogleChrome/webstatus.dev/tree/main/.dev/auth/" "Firebase Auth Emulator Setup"
    click FakeUsers "https://github.com/GoogleChrome/webstatus.dev/tree/main/util/cmd/load_test_users/" "Go to load_test_users Source"
    click FakeData "https://github.com/GoogleChrome/webstatus.dev/tree/main/util/cmd/load_fake_data/" "Go to load_fake_data Source"
    click RealFlows "https://github.com/GoogleChrome/webstatus.dev/tree/main/util/run_job.sh" "Go to run_job.sh"
```
