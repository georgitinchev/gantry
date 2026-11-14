export const site = {
  name: 'gantry',
  tagline: 'Infrastructure, requested by pull request.',
  repo: 'georgitinchev/gantry',
  branch: 'main',
  author: { name: 'Georgi Tinchev', url: 'https://georgitinchev.vercel.app' },
} as const;

export const repoUrl = `https://github.com/${site.repo}`;
export const fileUrl = (path: string) => `${repoUrl}/blob/${site.branch}/${path}`;
