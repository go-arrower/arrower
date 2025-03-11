import js from "@eslint/js";
import eslintConfigPrettier from "eslint-config-prettier";
import cypressConfig from "eslint-plugin-cypress/flat";
import globals from "globals";

export default [
  js.configs.recommended,
  cypressConfig.configs.recommended,
  eslintConfigPrettier,

  {
    ignores: [
      // ignore vendored js libraries
      "public/js/modules/",
    ],
  },
  {
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
    rules: {
      // "no-unused-vars": "warn",
      // "no-undef": "warn"
      "no-warning-comments": "error",
    },
  },
];
