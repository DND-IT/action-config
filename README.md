# action-config

Github Action for Managing configs

## usage

```yaml
- name: action-config
  uses: DND-IT/action-config@v1
```

## inputs

define your inputs in `action.yml`

## outputs

define your outputs in `action.yml`

## example

```yaml
name: example workflow
on: [push]

jobs:
  example:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: run action-config
        uses: DND-IT/action-config@v1
```

## development

edit `action.yml` to define your action's behavior.

this is a composite action, which means it uses shell scripts to execute logic.
you can use any language or tool you want by installing it in the composite steps.

## license

MIT

## support

Maintained by **group:default/dai**
