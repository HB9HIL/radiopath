# Radiopath documentation

Documentation for people who develop, run or extend Radiopath. It follows the [Diátaxis](https://diataxis.fr)
framework: every page is one of four kinds, and each kind answers a different need.

| | For learning | For working |
|---|---|---|
| **Practical steps** | [Tutorials](#tutorials): learn by doing, start to finish | [How-to guides](#how-to-guides): reach a specific goal |
| **Knowledge** | [Explanation](#explanation): understand why it is built this way | [Reference](#reference): look up exact facts |

## Tutorials

- [Getting started with development](tutorials/getting-started.md): from a fresh clone to a running dev stack and
  your first checked change.

## How-to guides

- [Add an API operation](how-to/add-an-api-operation.md)
- [Run and write end-to-end tests](how-to/run-and-write-e2e-tests.md)
- [Load terrain from S3](how-to/load-terrain-from-s3.md)
- [Build the tree canopy layer](how-to/build-the-canopy-layer.md)
- [Release a version](how-to/release-a-version.md)

## Reference

- [Configuration](reference/configuration.md): every environment variable
- [HTTP API](reference/http-api.md): conventions, errors, the routes next to the JSON API
- [Development commands and CI](reference/development.md): make targets, checks, CI jobs
- [Metrics and logging](reference/metrics-and-logging.md)
- [Operations](reference/operations.md): requirements, probes, memory, map tiles, user CLI

## Explanation

- [Architecture](explanation/architecture.md): how the binary, the API, the web app and the workers fit together
- [The propagation model](explanation/propagation-model.md): ITM, its limits, tree canopy, antenna patterns
- [Accounts, sharing and the admin panel](explanation/accounts-and-security.md)
- [Maps, legends and reports](explanation/maps-legends-and-reports.md)

## Writing documentation

Put a new page into the quadrant that matches what the reader wants, not the topic: a step-by-step guide to a goal
is a how-to even when it touches the API, a list of fields is reference even when it explains a little. Keep the
kinds apart and link between them instead of mixing them in one page. Add every new page to this index. The
contribution rules are in [CONTRIBUTING.md](../CONTRIBUTING.md).
