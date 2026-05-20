# Lux Desktop (shim)

Brand variant of [`hanzoai/desktop`](https://github.com/hanzoai/desktop).

This directory contains **zero source code**. It is a thin shim that builds Lux
Desktop from upstream `hanzo/desktop` source with the `BRAND=lux` overlay and
the brand assets in [`brand/`](./brand). It mirrors the
[`zoo/app`](https://github.com/zooai/app) and `zoo/exchange` patterns, so all
three brands ship the same machine / containers / Kubernetes / network UI from
one upstream codebase.

## Build

```bash
./scripts/build.sh
```

Produces a Lux-branded `.app` / `.dmg` at:

```
../../hanzo/desktop/apps/hanzo-desktop/src-tauri/target/release/bundle/
```

Override the upstream checkout with `HANZO_DESKTOP=/abs/path` if it isn't at
`../../hanzo/desktop` next to this repo.

## Dev

```bash
./scripts/dev.sh
```

Runs `tauri dev` against upstream with `VITE_BRAND=lux` so the app rebrands
live (logo, colors, identifier, the Machines / Containers / K8s / Network
panes from `libs/hanzo-machine-state`, `tauri.conf.lux.json` overlay).
