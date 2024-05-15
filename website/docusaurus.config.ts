import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  markdown: {
    mermaid: true,
  },
  themes: [
    '@docusaurus/theme-mermaid'
  ],
  
  title: 'Helmper Docs',
  tagline: 'A little helper that helps import OCI artifacts and Helm Charts to registries',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://christoffernissen.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/helmper/',

  trailingSlash: false,

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'ChristofferNissen', // Usually your GitHub org/user name.
  projectName: 'helmper', // Usually your repo name.

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   'https://github.com/ChristofferNissen/helmper/tree/main/',
        },
        blog: {
          showReadingTime: true,
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   'https://github.com/ChristofferNissen/helmper/tree/main/',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    image: 'img/docusaurus-social-card.jpg',
    navbar: {
      title: 'helmper',
      logo: {
        alt: 'Helmper Logo',
        src: 'img/helmper_logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'docs',
        },
        // {to: '/blog', label: 'Blog', position: 'left'},
        {to: '/what', label: 'what', position: 'left'},
        {to: '/how', label: 'how', position: 'left'},
        {to: '/why', label: 'why', position: 'left'},
        {
          href: 'https://github.com/ChristofferNissen/helmper',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Tutorial',
              to: '/docs/intro',
            },
            {
              label: 'Install',
              to: '/docs/install',
            },
          ],
        },
        // {
        //   title: 'Community',
        //   items: [
        //     // {
        //     //   label: 'Stack Overflow',
        //     //   href: 'https://stackoverflow.com/questions/tagged/docusaurus',
        //     // },
        //     // {
        //     //   label: 'Discord',
        //     //   href: 'https://discordapp.com/invite/docusaurus',
        //     // },
        //     // {
        //     //   label: 'Twitter',
        //     //   href: 'https://twitter.com/docusaurus',
        //     // },
        //   ],
        // },
        {
          title: 'More',
          items: [
            // {
            //   label: 'Blog',
            //   to: '/blog',
            // },
            {
              label: 'GitHub',
              href: 'https://github.com/ChristofferNissen/helmper',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} @ChristofferNissen.`,
    },
    prism: {
      additionalLanguages: ['bash'],
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
