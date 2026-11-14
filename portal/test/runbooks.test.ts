import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { parse } from 'yaml';
import { runbookFor } from '@/lib/snapshot';

const root = path.resolve(import.meta.dirname, '..', '..');
const rules = parse(readFileSync(path.join(root, 'observability', 'rules.yaml'), 'utf8'));
const alerts: { alert: string; annotations: { runbook_url: string } }[] = rules.spec.groups
  .flatMap((g: { rules: object[] }) => g.rules)
  .filter((r: { alert?: string }) => r.alert);

describe('every alert has a runbook', () => {
  it('found the alerts', () => expect(alerts.length).toBeGreaterThanOrEqual(3));

  for (const a of alerts) {
    it(`${a.alert}: runbook_url points at an existing file`, () => {
      const m = a.annotations.runbook_url.match(/\/blob\/main\/(docs\/runbooks\/[a-z-]+\.md)$/);
      expect(m, a.annotations.runbook_url).not.toBeNull();
      expect(existsSync(path.join(root, m![1]))).toBe(true);
    });
    it(`${a.alert}: portal links to the same runbook`, () => {
      const file = a.annotations.runbook_url.split('/').pop()!.replace(/\.md$/, '');
      expect(runbookFor(a.alert)).toBe(`/docs/runbooks/${file}`);
    });
  }
});
