# Contributing

**Requesting infrastructure:** add a YAML file under `requests/<team>/` and open a PR. See the [guides](docs/guides/). Run `make lint` first.

**Changing the platform:** open an issue first for anything beyond a small fix. Before pushing, run:

```bash
make test
make lint
make fresh
```

- Changes ship with tests: policy fixtures, promtool tests, or e2e checks.
- Pin versions: images by tag or digest, actions by SHA, tools in `hack/versions.env`.
- Update `docs/` in the same PR.
- One-line [Conventional Commits](https://www.conventionalcommits.org) subjects.

Security issues: see [SECURITY.md](SECURITY.md). Contributions are licensed under [Apache 2.0](LICENSE).
