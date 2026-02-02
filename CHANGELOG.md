# Changelog

All notable changes to this project will be documented in this file. See [Conventional Commits](https://conventionalcommits.org) for commit guidelines.

## [1.2.0](https://github.com/DND-IT/action-config/compare/v1.1.0...v1.2.0) (2026-02-02)

### ⚠ BREAKING CHANGES

* The legacy array format is no longer supported.
Configuration must be an object. Environments are now derived from
the keys of the "config" object, removing the need for a separate
"environments" array.

* fix: update CI workflow to use list-based config files

The test workflow still referenced the removed legacy array config
files. Updated all jobs to use valid-list-config.json/yml instead.

* feat: support exclude and include for matrix filtering

- exclude: array of patterns to remove from the cartesian product
- include: array of standalone entries appended to the matrix

This allows stacks that don't need environments to be handled via
exclude (filter specific combos) or include (add one-off entries).

* docs: update README for v2 format, exclude/include, drop legacy references

* feat: add stack, environment, exclude, and include action inputs

Adds workflow-level filtering inputs:
- stack/environment: comma-separated convenience filters
- exclude/include: JSON array inputs for advanced filtering

These are applied after the config file expansion, allowing
workflow_dispatch and reusable workflows to target subsets
without changing the config file.

### ✨ Features

* derive environments from config keys, drop legacy array format ([#3](https://github.com/DND-IT/action-config/issues/3)) ([e411389](https://github.com/DND-IT/action-config/commit/e411389db2b10003f87bf665a33d61ebbbe39f72))

## [1.1.0](https://github.com/DND-IT/action-config/compare/v1.0.0...v1.1.0) (2025-11-17)

### ✨ Features

* support lists: ([#2](https://github.com/DND-IT/action-config/issues/2)) ([a3ede9d](https://github.com/DND-IT/action-config/commit/a3ede9d90cf5325fd4ff5623cbc0ddee163a5c25))

## 1.0.0 (2025-11-06)

### ✨ Features

* implement config manager ([e525b63](https://github.com/DND-IT/action-config/commit/e525b63c419232d87f5c9886aad793e3c4a4b66b))
