const test = require("node:test");
const assert = require("node:assert/strict");

const { parseSimpleIconSlugs, searchSimpleIcons } = require("../lib/simple-icons-index");

test("parseSimpleIconSlugs extracts table rows", () => {
  const markdown = `
| Brand name | Brand slug |
| :--- | :--- |
| \`Ubuntu\` | \`ubuntu\` |
| \`Arch Linux\` | \`archlinux\` |
`;
  const items = parseSimpleIconSlugs(markdown);
  assert.equal(items.length, 2);
  assert.equal(items[0].name, "Ubuntu");
  assert.equal(items[0].slug, "ubuntu");
  assert.equal(items[1].slug, "archlinux");
});

test("searchSimpleIcons filters by query", () => {
  const items = [
    { name: "Ubuntu", slug: "ubuntu" },
    { name: "Arch Linux", slug: "archlinux" },
  ];
  const out = searchSimpleIcons(items, "arch", 20);
  assert.equal(out.length, 1);
  assert.equal(out[0].slug, "archlinux");
});
