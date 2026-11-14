import { describe, expect, it } from 'vitest';
import type { Offering } from '@/lib/catalog';
import { filePath, toYaml, validate } from '@/lib/request';

const env: Offering = {
  kind: 'AppEnvironment', plural: 'appenvironments', apiVersion: 'platform.gantry.internal/v1alpha1',
  namespaced: false, filePrefix: 'appenvironment', description: '', title: '', blurb: '', backedBy: '',
  fields: [
    { name: 'team', type: 'string', required: true },
    { name: 'size', type: 'enum', required: true, options: ['small', 'medium', 'large'] },
    { name: 'owners', type: 'list', required: false },
  ],
};
const pg: Offering = {
  ...env, kind: 'PostgresInstance', namespaced: true, filePrefix: 'postgres',
  fields: [
    { name: 'size', type: 'enum', required: true, options: ['small', 'medium'] },
    { name: 'storageGB', type: 'integer', required: false, min: 1, max: 20 },
  ],
};

describe('catalog requests', () => {
  it('builds the same paths gantry-lint expects', () => {
    expect(filePath(env, { team: 'checkout', name: 'checkout-dev', namespace: '', values: {} })).toBe('requests/checkout/appenvironment.yaml');
    expect(filePath(pg, { team: 'checkout', name: 'orders', namespace: 'checkout-dev', values: {} })).toBe('requests/checkout/postgres-orders.yaml');
  });
  it('renders typed YAML', () => {
    const y = toYaml(pg, { team: 'checkout', name: 'orders', namespace: 'checkout-dev', values: { size: 'small', storageGB: '5' } });
    expect(y).toContain('storageGB: 5\n');
    expect(y).toContain('namespace: checkout-dev');
  });
  it('mirrors the ownership and range rules', () => {
    const e = validate(pg, { team: 'checkout', name: 'orders', namespace: 'search-dev', values: { size: 'small', storageGB: '50' } });
    expect(e.namespace).toBeDefined();
    expect(e.storageGB).toBe('maximum 20');
    expect(validate(env, { team: 'checkout', name: 'search-dev', namespace: '', values: { size: 'small' } }).name).toMatch(/checkout-/);
  });
});

