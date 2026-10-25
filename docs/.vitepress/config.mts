import { defineConfig } from 'vitepress'

const base = '/gh-worktree/'

export default defineConfig({
  title: 'gh-worktree',
  description: 'A GitHub CLI extension that gives every pull request and branch its own git worktree.',
  base,
  cleanUrls: true,
  head: [['link', { rel: 'icon', type: 'image/svg+xml', href: `${base}logo.svg` }]],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Review Mode', link: '/guide/review-mode' },
      { text: 'Commands', link: '/guide/commands' },
    ],

    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Review Mode', link: '/guide/review-mode' },
          { text: 'Commands', link: '/guide/commands' },
          { text: 'Configuration', link: '/guide/configuration' },
        ],
      },
    ],

    socialLinks: [{ icon: 'github', link: 'https://github.com/utam0k/gh-worktree' }],

    search: { provider: 'local' },

    editLink: {
      pattern: 'https://github.com/utam0k/gh-worktree/edit/main/docs/:path',
    },

    footer: {
      message: 'Released under the MIT License.',
    },
  },
})
