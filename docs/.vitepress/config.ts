import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Whisk',
  description: 'Daemon-owned agent workspace for local and remote terminals.',
  base: '/whisk/',

  themeConfig: {
    siteTitle: 'Whisk',

    nav: [
      { text: 'Guide', link: '/' },
      { text: 'Agent Interface', link: '/agent-interface' },
      { text: 'Plugins', link: '/plugins' },
      { text: 'GitHub', link: 'https://github.com/phin-tech/whisk' },
    ],

    sidebar: [
      {
        text: 'Whisk',
        items: [
          { text: 'Overview', link: '/' },
          { text: 'Agent Interface', link: '/agent-interface' },
          { text: 'Browser CDP Evaluation', link: '/browser-cdp-evaluation' },
          { text: 'Plugins', link: '/plugins' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/phin-tech/whisk' },
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 Phin Tech',
    },

    editLink: {
      pattern: 'https://github.com/phin-tech/whisk/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
})
