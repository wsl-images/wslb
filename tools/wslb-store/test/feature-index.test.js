const test = require("node:test");
const assert = require("node:assert/strict");

const { parseFeatureRows, searchFeatures } = require("../lib/feature-index");

test("parseFeatureRows extracts feature table rows", () => {
  const html = `
    <table>
      <tr>
        <td class="tg-0lax"><a href="#">Node.js</a></td>
        <td class="tg-0lax">Dev Container Spec Maintainers</td>
        <td class="tg-0lax"><code>ghcr.io/devcontainers/features/node:1</code></td>
        <td class="tg-0lax"><code>1.7.1</code></td>
      </tr>
      <tr>
        <td class="tg-0lax"><a href="#">Python</a></td>
        <td class="tg-0lax">Maintainer Two</td>
        <td class="tg-0lax"><code>ghcr.io/devcontainers/features/python:1</code></td>
        <td class="tg-0lax"><code>1.8.0</code></td>
      </tr>
    </table>
  `;
  const rows = parseFeatureRows(html);
  assert.equal(rows.length, 2);
  assert.equal(rows[0].name, "Node.js");
  assert.equal(rows[0].ref, "ghcr.io/devcontainers/features/node:1");
  assert.equal(rows[1].maintainer, "Maintainer Two");
});

test("searchFeatures filters by query", () => {
  const items = [
    { name: "Node.js", maintainer: "A", ref: "ghcr.io/devcontainers/features/node:1", version: "1.7.1" },
    { name: "Python", maintainer: "B", ref: "ghcr.io/devcontainers/features/python:1", version: "1.8.0" },
  ];
  const out = searchFeatures(items, "python", 10);
  assert.equal(out.length, 1);
  assert.equal(out[0].name, "Python");
});
