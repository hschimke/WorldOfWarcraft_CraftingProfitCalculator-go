# WorldOfWarcraft_CraftingProfitCalculator (CPC) - Context

This project is a World of Warcraft Crafting Profit Calculator, designed to evaluate the profitability of crafting items compared to their current auction house and vendor prices.

## Project Overview

-   **Backend:** Go (Golang) using `pgx` for PostgreSQL, `go-redis` for Redis caching, and custom implementations for Blizzard API interaction.
-   **Frontend:** React with TypeScript, using `react-scripts`.
-   **Data Storage:** PostgreSQL for auction history and item metadata; Redis for short-term caching.
-   **Integration:** Multi-stage Docker deployment that bundles the React frontend and Go backend into a single container.
-   **Data Sources:** Blizzard API (for auctions, items, and recipes) and local JSON files (`static_files/`) for game-specific mappings (bonuses, ranks, etc.).
-   **Character Data:** A Lua addon (`wow-addon/`) allows users to export character inventory and professions for more accurate local profit calculations.

## Architecture & Components

-   `cmd/`: Main entry points for various CLI tools and background workers.
    -   `WorldOfWarcraft_CraftingProfitCalculator-go`: Primary CLI for profit calculations.
    -   `auction_archive_ctrl`: Utility for managing auction history and scanning tasks.
    -   `hourly_injest`: Background worker for ingesting auction and item data.
    -   `run_worker`: Background worker that processes jobs for the web interface.
-   `internal/`: Internal Go packages (not intended for external use).
    -   `blizz_oath`: Handles Blizzard API OAuth2 token management.
    -   `blizzard_api_call`: Low-level Blizzard API provider.
    -   `cache_provider`: Redis-based caching logic.
    -   `cpclog`: Custom logging wrapper.
    -   `environment_variables`: Centralized configuration management using environment variables.
    -   `routes`: API route definitions for the web server.
    -   `static_sources`: Logic for reading project-specific static data files.
-   `pkg/`: Reusable Go packages.
    -   `wow_crafting_profits`: Core logic for profit analysis and shopping list construction.
    -   `blizzard_api_helpers`: Higher-level abstractions for Blizzard API data.
    -   `globalTypes`: Shared types used across the codebase.
-   `web-serv/`: Implementation of the web server that serves both the API and the React frontend.
-   `web-client/`: The React-based user interface.
-   `static_files/`: JSON configuration data for game-specific mappings.
-   `wow-addon/`: Lua-based World of Warcraft addon for inventory export.

## Development & Operations

### Requirements

-   **PostgreSQL:** Required for persistent data.
-   **Redis:** Required for caching.
-   **Blizzard API Credentials:** `CLIENT_ID` and `CLIENT_SECRET` must be obtained from the Blizzard Developer Portal.

### Key Commands

-   **Full Build (Docker):**
    ```bash
    sh scripts/docker-build.sh
    ```
-   **Run Backend (Local):**
    Requires environment variables (`CLIENT_ID`, `CLIENT_SECRET`, `DATABASE_CONNECTION_STRING`, `REDIS_URL`).
    ```bash
    go build ./web-serv/WorldOfWarcraft_CraftingProfitCalculator-go/ && ./WorldOfWarcraft_CraftingProfitCalculator-go
    ```
-   **Frontend (Dev):**
    ```bash
    cd web-client && npm start
    ```
-   **Testing:**
    -   Backend: `go test ./...` (Note: Some tests require `CLIENT_ID` and `CLIENT_SECRET` in the environment).
    -   Frontend: `npm test` in `web-client`.

### Configuration (Environment Variables)

-   `CLIENT_ID`: Blizzard API Client ID.
-   `CLIENT_SECRET`: Blizzard API Client Secret.
-   `DATABASE_CONNECTION_STRING`: PostgreSQL connection string.
-   `REDIS_URL`: Redis connection string.
-   `SERVER_PORT`: Port for the web server (defaults to `3001`).
-   `STANDALONE_CONTAINER`: Defines the worker mode (`normal`, `hourly`, `worker`, `standalone`).
-   `STATIC_DIR_ROOT`: Root directory for static data files.

## Development Conventions

-   **Log Style:** Uses a custom `cpclog` package with levels (`info`, `debug`, `error`, `silly`).
-   **Static Data:** Game data is often stored in `static_files/*.json` and accessed through the `static_sources` package.
-   **Docker:** Uses a multi-stage Dockerfile (`docker/Dockerfile.web-serv`) to build the entire stack.
-   **Internal/Package Separation:** High-level calculation logic is in `pkg/wow_crafting_profits`, while infrastructure concerns are in `internal/`.
