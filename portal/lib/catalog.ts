import 'server-only';
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { parseAllDocuments } from 'yaml';

/** One input on a catalog form, derived from the XRD's OpenAPI schema. */
export type Field = {
  name: string;
  type: 'enum' | 'string' | 'integer' | 'list';
  description?: string;
  required: boolean;
  options?: string[];
  min?: number;
  max?: number;
  pattern?: string;
  default?: string | number;
};

export type Offering = {
  kind: string;
  plural: string;
  apiVersion: string;
  namespaced: boolean;
  filePrefix: string;
  description: string;
  fields: Field[];
  title: string;
  blurb: string;
  backedBy: string;
};

// Presentation only; everything structural comes from the XRDs.
const presentation: Record<string, Pick<Offering, 'title' | 'blurb' | 'backedBy'>> = {
  AppEnvironment: {
    title: 'App environment',
    blurb: 'An isolated namespace with quota, limits, default-deny networking and owner RBAC.',
    backedBy: 'Namespace · ResourceQuota · LimitRange · NetworkPolicy · RoleBinding',
  },
  PostgresInstance: {
    title: 'PostgreSQL',
    blurb: 'A managed Postgres cluster inside one of your environments, reachable only from it.',
    backedBy: 'CloudNativePG Cluster · NetworkPolicy',
  },
  VirtualMachine: {
    title: 'Virtual machine',
    blurb: "A VM in gantry-cloud, placed in your environment's private network. Resizes in place.",
    backedBy: 'gantry-cloud Instance · via provider-gantry-cloud',
  },
};

type Schema = {
  type?: string;
  description?: string;
  enum?: string[];
  pattern?: string;
  minimum?: number;
  maximum?: number;
  default?: string | number;
  required?: string[];
  properties?: Record<string, Schema>;
};

const apisDir = () => path.join(process.cwd(), '..', 'platform', 'apis');

export async function getCatalog(): Promise<Offering[]> {
  const dirs = await readdir(apisDir(), { withFileTypes: true });
  const offerings: Offering[] = [];
  for (const d of dirs.filter((d) => d.isDirectory())) {
    const raw = await readFile(path.join(apisDir(), d.name, 'definition.yaml'), 'utf8');
    for (const doc of parseAllDocuments(raw)) {
      const xrd = doc.toJS();
      if (xrd?.kind !== 'CompositeResourceDefinition') continue;
      const version = xrd.spec.versions.find((v: { served: boolean }) => v.served);
      const schema: Schema = version.schema.openAPIV3Schema;
      const spec = schema.properties?.spec ?? {};
      const kind: string = xrd.spec.names.kind;
      offerings.push({
        kind,
        plural: xrd.spec.names.plural,
        apiVersion: `${xrd.spec.group}/${version.name}`,
        namespaced: xrd.spec.scope !== 'Cluster',
        filePrefix: xrd.metadata.annotations['platform.gantry.internal/file-prefix'],
        description: schema.description ?? '',
        fields: toFields(spec),
        ...(presentation[kind] ?? { title: kind, blurb: schema.description ?? '', backedBy: '' }),
      });
    }
  }
  return offerings.sort((a, b) => Number(a.namespaced) - Number(b.namespaced));
}

function toFields(spec: Schema): Field[] {
  const required = new Set(spec.required ?? []);
  return Object.entries(spec.properties ?? {}).map(([name, s]) => {
    const base = { name, description: s.description, required: required.has(name), default: s.default };
    if (s.enum) return { ...base, type: 'enum', options: s.enum };
    if (s.type === 'integer') return { ...base, type: 'integer', min: s.minimum, max: s.maximum };
    if (s.type === 'array') return { ...base, type: 'list' };
    return { ...base, type: 'string', pattern: s.pattern };
  });
}
