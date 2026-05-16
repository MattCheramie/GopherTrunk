// Vitest setup file. Loaded before every test file via
// `test.setupFiles` in vite.config.ts. Wires in
// @testing-library/jest-dom's assertion vocabulary
// (`toBeInTheDocument`, `toHaveClass`, etc.) and the cleanup
// behaviour that unmounts components between tests.

import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => {
  cleanup();
});
