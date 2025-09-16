---
order: 5
metaTitle: "App Template"
footer:
  newsletter: false
---

# Quickstart: App Template

The `NibiJS` App Template is designed to jumpstart your frontend development with `NibiJS`. This template is built using Next.js, React, Cosmos-kit, and Tailwind CSS, providing a robust foundation to accelerate your development process.

#### Table of Contents

- [Usage Instructions](#usage-instructions)
  - [Install Node Version Manager (`nvm`)](#install-node-version-manager-nvm)
  - [Run the application locally](#run-the-application-locally)
- [Dependencies](#dependencies)
- [Source (src) Directory Structure](#source-src-directory-structure)
- [Deployments](#deployments)
- [Related Pages](#related-pages)


## Usage Instructions

### Install Node Version Manager (`nvm`)

To install or update nvm, run the install script using the following command:

```bash
wget -qO- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
```

For more information, see [github.com/nvm-sh/nvm](https://github.com/nvm-sh/nvm).

### Run the application locally

1. Move to the repo's node version as defined by the .nvmrc:

   ```bash
   nvm use
   ```

2. Install yarn and download package dependencies (can take ≈3 minutes):

   ```bash
   npm install -g yarn
   yarn # or: npm install
   ```

3. Run the application with:

   ```bash
    npm run dev   # defaults to http://localhost:3000
    # or
    yarn dev
    # or
    pnpm dev
    # or
    bun dev
   ```

    This will automatically open an interactive development environment with hot
    reloading, enabling you to edit files and see changes reflected in the
    running application.

    Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

    You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

## Dependencies

- [Nibi JS](https://github.com/NibiruChain/ts-sdk)
- [Next js](https://nextjs.org/)
- [React js](https://react.dev/)
- [Tailwind](https://tailwindcss.com/)
- [Cosmos-Kit](https://cosmology.zone/products/cosmos-kit)

## Source (src) Directory Structure

- `components`: Contains all reusable components.
- `pages`: A directory containing the page-level components. Each subdirectory of `pages` corresponds to a page in the application.
- `layouts`: For layout-based components like sidebars, navbars, containers, page headers, and page footers.
- `hooks`: For custom hooks.
- `context`: Contains logic related to the global Redux store.
- `config`: For custom types re-used in multiple places. A subdirectory or file named "types" exports local to a specific directory.
- `style`: For utility functions that didn't fit into other categories.

## Deployments

After building, you can upload `dist` folder to a hosting service like Netlify

```bash
yarn build
```

## Related Pages

- [NibiJS Getting Started](./getting-started.md)
- [NibiJS Connecting wiht a wallet extension](./connect-wallet.md)
- [NibiJS Smart Contracts](./smart-contracts.md)
- [NibiJS Querier](./querier.md)
