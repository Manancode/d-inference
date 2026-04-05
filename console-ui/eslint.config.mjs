import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";

const eslintConfig = defineConfig([
  ...nextVitals,
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts"]),
  {
    rules: {
      // React 19.2 adds this rule but our init-in-useEffect patterns are fine
      // (theme init from localStorage, settings load, etc.)
      "react-hooks/set-state-in-effect": "warn",
    },
  },
]);

export default eslintConfig;
