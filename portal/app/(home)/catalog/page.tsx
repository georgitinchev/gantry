import type { Metadata } from 'next';
import { CatalogBrowser } from '@/components/catalog-browser';
import { getCatalog } from '@/lib/catalog';

export const metadata: Metadata = {
  title: 'Catalog',
  description: 'Request platform resources. Forms are generated from the live API definitions.',
};

export default async function CatalogPage() {
  const offerings = await getCatalog();
  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
      <p className="annot">Service catalog</p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight sm:text-4xl">What do you need?</h1>
      <p className="mt-3 max-w-2xl text-ink-2">
        Every form below is generated from the platform&apos;s CompositeResourceDefinitions, so it can never drift from the
        real API. Fill one in and open it as a pull request. CI does the rest.
      </p>
      <CatalogBrowser offerings={offerings} />
    </main>
  );
}
