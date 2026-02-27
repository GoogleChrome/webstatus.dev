# Webstatus.dev Architecture

This document contains high-level architectural diagrams for the webstatus.dev ecosystem. These diagrams are intended to help new developers understand the system boundaries, data flows, and local development environment.

```mermaid
graph LR
    BCD[Browser Compat Data] --> WebStatus((webstatus.dev))
    WPT[Web Platform Tests] --> WebStatus
    UMA[Chromium UMA] --> WebStatus
    GH[GitHub Auth / Profiles] --> WebStatus
    Other[Other data sources] --> WebStatus
    WebStatus --> Devs[Web Developers]
    WebStatus --> Vendors[Browser Vendors]
```

---

## Functional Architecture (Core Blocks)

This diagram identifies the major system boundaries and their primary responsibilities.

```mermaid
graph TD
    subgraph Ingestion ["1. Ingestion"]
        BCD[BCD Consumer]
        WPT[WPT Consumer]
        UMA[UMA Export]
    end

    subgraph State ["2. State & Cache"]
        DB[(Spanner: Source Table)]
        Valkey[(Valkey: Cache Layer)]
    end

    subgraph Backend ["3. Backend API"]
        Handlers[Go API Handlers]
    end

    subgraph Auth ["4. Authentication"]
        Firebase[Firebase Auth]
        GH[GitHub OAuth]
    end

    subgraph Frontend ["5. Frontend SPA"]
        Lit[Lit Components]
    end

    subgraph Notifications ["6. Notifications"]
        EP[Event Producer]
        Disp[Dispatcher]
        Workers[Delivery Workers]
    end

    Ingestion -->|Syncs to| DB
    DB <-->|Reads/Writes| Handlers
    Handlers <-->|JSON API| Lit
    Lit <-->|Identity| Auth
    Handlers <-->|Verify JWT| Auth
    DB -->|Triggers| EP
    EP -->|Fans-out| Disp
    Disp -->|Jobs to| Workers
```

---

For a detailed database schema overview, see the [Generated Spanner Documentation](schema/README.md).

_Note: The nodes in these diagrams are clickable! Click on a component to jump to its source code or relevant GCP documentation._

---

## 1. High-Level Architecture (Multi-Project Setup)

To limit the scope of the `allUsers` IAM permission (which makes the frontend and API public), the infrastructure is split across three GCP projects. This **Security Isolation** ensures that even if a public-facing service is compromised, the primary database and ingestion state remain protected behind VPC Service Controls and Shared VPC networking.

### Core Ecosystem Entities

Before diving into the diagrams, it is helpful to understand the core domain entities:

- **Feature**: The atomic unit of tracking (e.g., "WebAssembly", "CSS Grid"). Derived from the `web-features` repo.
- **Run**: A specific execution of Web Platform Tests for a browser/version.
- **Metric**: Passing/Failing stats (from WPT) or real-world usage percentages (from UMA).
- **Snapshot**: A GCS-stored record of search results at a point in time, used for cross-job diffing.
- **Saved Search**: A persistent user query (e.g., "Baseline: Widely") that triggers notifications.
- **Notification Channel**: A delivery target for alerts (Email or Web Push).

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

### Authentication & User Identity

User accounts and identity are managed via **GitHub OAuth**, bridged by **Firebase Auth**.

> [!TIP]
> **Implementation Details**: For technical details on JWT verification middleware and frontend login handshake logic, refer to the [Backend Architecture Guide](../skills/webstatus-backend/references/architecture.md) and [Frontend Architecture Guide](../skills/webstatus-frontend/references/architecture.md).

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
    U->>FE: Login with GitHub
    FE->>BE: GET /v1/users/me/saved-searches (Auth: Bearer JWT)
    Note over BE, C: Check Cache (Only for public stats)
    BE->>C: Lookup
    alt Cache Hit
        C-->>BE: Return JSON
    else Cache Miss / Bypass
        Note right of BE: Always bypass cache for Auth'd User Data
        BE->>DB: Query via Spanner Adapter (Filter by UserID)
        DB-->>BE: Return Rows
        BE->>C: Optional Cache Update (Stats only)
    end
    BE-->>FE: HTTP 200 JSON
    FE-->>U: Render Personalized UI
```

> [!TIP]
> **Implementation Details**: For the specific Go handlers, ANTLR parser technicalities, and caching code, refer to the [Backend Implementation Guide](../skills/webstatus-backend/references/architecture.md) and [Search Grammar Guide](../skills/webstatus-search-grammar/references/architecture.md).

---

## 3. Storage & Infrastructure (Verified Setup)

The project leverages managed GCP services, networked for isolation in the `internal` project.

### Service Connectivity Matrix

| Component        | GCP Service          | Connectivity Pattern                  | Terraform Link                              |
| :--------------- | :------------------- | :------------------------------------ | :------------------------------------------ |
| **Database**     | Spanner              | Public API (via IAM/VPC-SC)           | [spanner.tf](../infra/storage/spanner.tf)   |
| **Cache**        | Memorystore (Valkey) | **Private Service Connect (PSC)**     | [valkey.tf](../infra/storage/valkey.tf)     |
| **Managed APIs** | Google Services      | **VPC Peering** (`servicenetworking`) | [network/main.tf](../infra/network/main.tf) |
| **Storage**      | GCS                  | Authenticated API                     | [storage/](../infra/storage/)               |

> [!IMPORTANT]
> **VPC Peering vs PSC**: We use VPC Peering for general Google Managed Services communication, but **Private Service Connect (PSC)** is specifically used for the Valkey instance to provide dedicated, secure endpoints within our subnets.

## 4. Data Ingestion Pipeline (Internal Project)

Webstatus.dev relies heavily on external data. Cloud Scheduler triggers jobs that download, parse, and synchronize this data into our Spanner database.

```mermaid
flowchart LR
        subgraph External ["External Sources"]
            WPT["WPT.fyi API"]
            BCD["MDN BCD Repo"]
            WF["Web-Features Repo"]
            WFM["WF Mappings Repo"]
            UMA["Chromium UMA"]
        end

    subgraph Internal ["Project: Internal"]
        direction TB
        Sched["Cloud Scheduler"] --> Ingestion_Jobs

        subgraph Ingestion_Jobs ["Ingestion Services (Go)"]
            BCD_C["BCD Consumer"]
            WPT_C["WPT Consumer"]
            WF_C["Web-Features Consumer"]
            WFM_C["Mapping Consumer"]
            DS_C["Dev Signals Consumer"]
            UMA_C["UMA/Histogram Consumer"]
            HE_C["Histogram Enum Consumer"]
        end

        subgraph Job_Logic ["Internal Logic (Service-Specific)"]
            DL["Custom Downloader"] --> P["Custom Parser"]
            P --> Adp["Spanner Adapter / Mappers"]
        end

        Adp --> DB[(Cloud Spanner)]
    end

    WPT --> WPT_C
    BCD --> BCD_C
    WF --> WF_C
    WFM --> WFM_C
    UMA --> UMA_C

    Ingestion_Jobs -.-> DL

    %% Clickable Links
    click BCD_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/bcd_consumer/" "BCD Consumer Source"
    click WPT_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/wpt_consumer/" "WPT Consumer Source"
    click WF_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/web_feature_consumer/" "Web-Features Source"
    click WFM_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/web_features_mapping_consumer/" "Mapping Consumer Source"
    click DS_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/developer_signals_consumer/" "Dev Signals Source"
    click UMA_C "https://github.com/GoogleChrome/webstatus.dev/tree/main/workflows/steps/services/uma_export/" "UMA Export Source"
    click DB "https://github.com/GoogleChrome/webstatus.dev/blob/main/docs/schema/README.md" "Spanner Schema Overview"
```

> [!TIP]
> **Implementation Details**: For the full list of data source schemas and job orchestration patterns, refer to the [Ingestion Implementation Guide](../skills/webstatus-ingestion/references/architecture.md).

---

## 5. Notification System Architecture (Event-Driven)

When ingestion jobs complete, the system triggers a fan-out process to notify users. This architecture strictly separates **Change Detection** (Delta) from **Subscription Matching** (Dispatch) and **Delivery** (Worker).

> [!NOTE]
> **Push vs On-Demand Pull**: While most notifications are pushed (Email, Webhooks), the system is designed to also support **On-Demand Pull** channels (e.g., RSS, API Feeds) one day. The general idea: these are still subscription-bound but bypass the push dispatcher, instead being served via the API layer when requested by the client.

> [!TIP]
> **Technical Deep-Dive**: For code-level implementation choreography (struct types, differ logic, and versioning patterns), refer to the [Worker Architecture Guide](../skills/webstatus-workers/references/architecture.md) in the workers skill. This document focuses on the high-level system topology.

```mermaid
flowchart TD
    %% Component Definitions
    subgraph Infra ["Infrastructure & Storage"]
        DB[(Spanner: Subscriptions)]
        GCS[(GCS: Snapshots)]
    end

    subgraph Topics ["Events (Pub/Sub)"]
        T1{{batch-updates}}
        T2{{delivery-events}}
    end

    subgraph Workers ["Workflow Workers<br/>(Cloud Run Jobs)"]
        direction TB
        w_spacer[ ]
        EP[Event Producer]
        Disp[Notification Dispatcher]
        EW[Email Worker]
        WH[Webhook Worker]
    end
    style w_spacer fill:none,stroke:none,color:none
    w_spacer ~~~ EP

    subgraph API ["Runtime Services"]
        Backend[Backend API]
        RSS["RSS Feed (On-Demand)"]
    end

    subgraph Libs ["Libraries & External"]
        Chime[External: Chime Service]
        ExtWH[External: Webhook URL]
    end

    %% Flow
    Ingestion[(Ingestion Complete)] --> EP
    Backend -->|Search Updated| Disp
    EP -->|1. Diff against| GCS
    EP -->|2. Publish Diffs| T1

    T1 --> Disp
    Disp -->|3. Find Subscribers| DB
    Disp -->|4. Fan-out per user| T2

    T2 --> EW
    T2 -.->|In-Progress| WH

    EW -->|5. Render & Send| Chime
    WH -.->|POST JSON| ExtWH

    Backend -.->|Pull Feed| RSS

    %% Styling
    classDef topic fill:#312e81,stroke:#818cf8,color:#fff,stroke-width:2px;
    classDef worker fill:#1e3a8a,stroke:#60a5fa,color:#fff,stroke-width:2px;
    classDef infra fill:#1f2937,stroke:#9ca3af,color:#fff,stroke-width:2px;
    class T1,T2 topic;
    class EP,Disp,EW,WH worker;
    class DB,GCS infra;
```

#### Nuance: Email Delivery Adapters

Currently, the system uses the [Chime Adapter](../lib/email/chime) for production delivery. However, the architecture is designed to support an **SMTP Adapter** in the future. This would allow developers to point local workers at a tool like Mailhog or Mailtrap for end-to-end testing without needing a real Chime environment.

#### Notification Component Roles

| Type        | Name                      | Purpose                                                                                    |
| :---------- | :------------------------ | :----------------------------------------------------------------------------------------- |
| **Worker**  | `event_producer`          | Detects deltas between current Spanner state and last GCS snapshot.                        |
| **Event**   | `batch-updates`           | High-level event containing the search ID and a summary of what changed.                   |
| **Worker**  | `notification_dispatcher` | The "Fan-Out" engine. Finds all subscribers for a search and creates unique delivery jobs. |
| **Event**   | `delivery-events`         | Individual job for a specific user and channel (Go Canonical Type).                        |
| **Worker**  | `email_worker`            | Renders templates using `lib/email` and delivers via the Chime adapter.                    |
| **Library** | `lib/email`               | Shared HTML templates and Go mappers for email rendering.                                  |

---

> [!TIP]
> **Implementation Details**: For the technical deep-dives into Event Producer diffing, Dispatcher fan-out, and Email rendering logic, refer to the [Worker Implementation Guide](../skills/webstatus-workers/references/architecture.md).

> [!NOTE]
> **Implementation Details**: For an in-depth look at how versioned blobs are parsed using the `SummaryVisitor` pattern, refer to the [Worker Implementation Guide](../skills/webstatus-workers/references/architecture.md#3-schema-evolution--the-summaryvisitor).

---

---

## 6. Local Development & E2E Testing Environment

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
            WM["Wiremock<br/>(Mock GitHub API)"]
            Valkey[(Valkey Cache)]
            PS[[Pub/Sub Emulator]]
        end

        FE <-->|API Calls| BE
        FE <-->|Firebase Login| Auth
        BE <-->|Read/Write| Spanner
        BE <-->|Cache| Valkey
        BE <-->|Fetch Profile/Emails<br/>on Login| WM
    end

    subgraph Data_Population ["Data Population Strategies"]
        direction TB
        FakeUsers["make dev_fake_users<br/>(util/cmd/load_test_users)"]
        FakeData["make dev_fake_data<br/>(util/cmd/load_fake_data)"]
        RealFlows["make dev_workflows<br/>(util/run_job.sh)"]
    end
```

> [!TIP]
> **Implementation Details**: For a deep-dive into the local dev loop, E2E population scripts, and the CI gate logic, refer to the [E2E & CI Architecture Guide](../skills/webstatus-e2e/references/architecture.md).

---

### Path to Production (The PR Lifecycle)

Every contribution follows a strict automated validation path before it can be merged into `main`.

```mermaid
graph TD
    Dev[Local Commit] -->|Pull Request| CI[GitHub Action]
    subgraph Concurrent_Checks ["Concurrent CI Gates"]
        PC["make precommit"]
        PW["make playwright-test (E2E tests)"]
    end
    CI --> PC
    CI --> PW
    PC --> |Both Pass| Final[Approve & Merge]
    PW --> |Both Pass| Final
    Final --> Main[Main Branch]
```

---

## 7. Developer Guides & Patterns

For technical patterns and implementation details, refer to the **Gemini Skills** in your IDE:

- **[webstatus-backend](../skills/webstatus-backend/SKILL.md)**: Go API, Spanner mappers, and OpenAPI.
- **[webstatus-frontend](../skills/webstatus-frontend/SKILL.md)**: Lit web components and component testing.
- **[webstatus-e2e](../skills/webstatus-e2e/SKILL.md)**: Playwright E2E testing and debugging.
- **[webstatus-ingestion](../skills/webstatus-ingestion/SKILL.md)**: Scheduled data ingestion workflows.
- **[webstatus-workers](../skills/webstatus-workers/SKILL.md)**: Pub/Sub notification pipeline.
- **[webstatus-search-grammar](../skills/webstatus-search-grammar/SKILL.md)**: ANTLR search query parsing.
- **[webstatus-maintenance](../skills/webstatus-maintenance/SKILL.md)**: Toolchain upgrades and infra maintenance.

---

## 8. Local Environment

Since we use a **Devcontainer**, all engineers get a pre-configured environment with built-in tools for testing and debugging.

- **[docs/debugging.md](debugging.md)**: Using the ANTLR visualizer and attaching debuggers to Go unit tests.
- **[docs/testing.md](testing.md)**: Documentation for Playwright iteration, `make precommit`, and VS Code Test Explorer tips.
