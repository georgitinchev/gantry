import { defineConfig, defineDocs } from 'fumadocs-mdx/config';
import { remarkMdxMermaid } from 'fumadocs-core/mdx-plugins';

// Docs live at the repository root so they render on GitHub too.
export const docs = defineDocs({ dir: '../docs' });

export default defineConfig({
  mdxOptions: { remarkPlugins: [remarkMdxMermaid] },
});
