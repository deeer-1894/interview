import { promises as fs } from "node:fs";
import path from "node:path";

const projectRoot = process.cwd();
const outDir = path.join(projectRoot, ".test-dist", "src");

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath));
      continue;
    }
    if (entry.isFile() && fullPath.endsWith(".js")) {
      files.push(fullPath);
    }
  }
  return files;
}

function withJsExtension(specifier) {
  if (/\.(js|mjs|cjs|json)$/i.test(specifier)) {
    return specifier;
  }
  return `${specifier}.js`;
}

function rewriteSpecifier(filePath, specifier) {
  if (specifier.startsWith("@/")) {
    const target = withJsExtension(path.join(outDir, specifier.slice(2)));
    let relative = path.relative(path.dirname(filePath), target).replaceAll(path.sep, "/");
    if (!relative.startsWith(".")) {
      relative = `./${relative}`;
    }
    return relative;
  }

  if (specifier.startsWith("./") || specifier.startsWith("../")) {
    return withJsExtension(specifier);
  }

  return specifier;
}

async function rewriteImports(filePath) {
  const source = await fs.readFile(filePath, "utf8");
  const updated = source.replace(/(from\s+|import\s*\()(["'])([^"']+)(["'])/g, (match, prefix, quote, specifier, suffixQuote) => {
    const rewritten = rewriteSpecifier(filePath, specifier);
    return `${prefix}${quote}${rewritten}${suffixQuote}`;
  });

  if (updated !== source) {
    await fs.writeFile(filePath, updated);
  }
}

async function main() {
  const files = await walk(outDir);
  await Promise.all(files.map(rewriteImports));
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
