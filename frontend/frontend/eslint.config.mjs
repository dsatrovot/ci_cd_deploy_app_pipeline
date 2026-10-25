import js from "@eslint/js";
import globals from "globals";

export default [
  {
    files: ["**/*.js", "**/*.mjs"],
    ignores: ["dist/**", "node_modules/**"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: globals.browser
    },
    rules: {
      ...js.configs.recommended.rules
    }
  }
];