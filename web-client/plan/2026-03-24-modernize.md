# Modernization Plan: React 19.2.4 & TypeScript 6.0.0 (March 24, 2026)

## 1. Executive Summary
This plan outlines the steps required to modernize the `web-client` from a legacy Create React App (CRA) environment to a high-performance, modern React 19 ecosystem. This includes migrating to Vite, upgrading to the newly released TypeScript 6.0, and adopting the React Compiler.

## 2. Current vs. Target State
| Feature | Current State | Target State |
| :--- | :--- | :--- |
| **Framework** | React 18.2.0 | React 19.2.4 |
| **Language** | TypeScript 4.7.4 | TypeScript 6.0.0 |
| **Build Tool** | CRA (react-scripts) | Vite 6.x |
| **Optimization** | Manual (useMemo/useCallback) | React Compiler (Automated) |
| **Form Handling** | useReducer / Custom Hooks | useActionState / useFormStatus |
| **API Integration** | Custom Fetch Wrapper | React `use()` + Standard Promises |

## 3. Phase 1: Infrastructure & Tooling (Vite Migration)
*   **Action:** Replace `react-scripts` with Vite.
*   **Rationale:** CRA is deprecated and incompatible with the performance and feature set of React 19 / TS 6.0.
*   **Steps:**
    1. Uninstall `react-scripts`.
    2. Install `vite`, `@vitejs/plugin-react`, and `vite-tsconfig-paths`.
    3. Move `public/index.html` to root and update script tags (Vite expects the entry point in the root).
    4. Rename `.env` variables from `REACT_APP_*` to `VITE_*`.
    5. Update `package.json` scripts to use `vite`.
    6. Configure `vite.config.ts` with the existing proxy (`http://localhost:3001/`).

## 4. Phase 2: TypeScript 6.0 "Bridge" Migration
*   **Action:** Upgrade to the final JS-based TS compiler before the TS 7.0 (Go) rewrite.
*   **Rationale:** TS 6.0 removes support for legacy module resolution and targets, requiring a cleanup of `tsconfig.json`.
*   **Steps:**
    1. Update `typescript` to `^6.0.0` and `@types/react*` to version 19 equivalents.
    2. Refactor `tsconfig.json`:
        *   Change `moduleResolution` to `bundler`.
        *   Update `target` to `ES2025`.
        *   Enable `verbatimModuleSyntax` (replacing `importsNotUsedAsValues`).
        *   Remove support for `node10` resolution and `es5` target.
    3. Resolve new strictness errors in `ApiClient.ts` (e.g., removing `any` in favor of generics) and `RegionSelector.tsx`.
    4. Implement the **Temporal API** for date handling in `Auctions.tsx` (now native in TS 6.0).

## 5. Phase 3: React 19 Features & The Compiler
*   **Action:** Adopt the React Compiler and modern data patterns.
*   **Steps:**
    1. Update `react` and `react-dom` to `^19.2.4`.
    2. Install the React Compiler Vite plugin.
    3. **The Great Deletion:** Scan `Auctions.tsx` and `RunCoordinator.tsx` to remove manual `useMemo` and `useCallback` calls. The compiler now handles this automatically.
    4. Refactor `RunResultDisplay.tsx`: Replace the custom `fetchPromiseWrapper` with standard promises consumed via the native `use()` hook.
    5. Update Forms: Migrate `RunForm.tsx` to use `useActionState` and `useFormStatus` for handling submission states and pending indicators.

## 6. Phase 4: Accessibility & Cleanup
*   **Action:** Address technical debt and UI polish.
*   **Steps:**
    1. Refactor `AutoCompleteBox.tsx` and `RegionSelector.tsx` for full keyboard navigation (arrows/enter) and ARIA support.
    2. Remove legacy `console.log` and `false && ...` debugging blocks from `Auctions.tsx` and `BonusListDropdown.tsx`.
    3. Update `Links.tsx`:
        *   Change copyright to `new Date().getFullYear()`.
        *   Fix typographical errors: "Source Coce", "let us known".
    4. Standardize all components to `function` declarations (removing the mix of `const` components).

## 7. Verification & Testing
*   **Type Check:** Run `tsc --noEmit` to ensure TS 6.0 compliance.
*   **Build:** Run `npm run build` to verify the ES2025 bundle and Vite optimization.
*   **Verification:** Confirm the Go backend correctly receives `POST` requests from the new Vite proxy.

## 8. Estimated Timeline
*   **Day 1:** Phase 1 (Vite & Build Tooling).
*   **Day 2:** Phase 2 (TypeScript 6.0 Cleanup & Strictness).
*   **Day 3-4:** Phase 3 (React 19 & Compiler Adoption).
*   **Day 5:** Phase 4 (Accessibility, Typos, & Final Verification).
