---
order: 1
metaTitle: "Install NibiJS using npm, yarn, or bun"
---

# Installation

Developing on the Nibiru is made easy with NibiJS. This TypeScript library simplifies decentralized application (dApp) development, providing a user-friendly interface that abstracts away blockchain complexities. With NibiJS, developers can focus on innovation and building cutting-edge dApps.


To install @nibiruchain/nibijs, follow these steps. This guide assumes you have a basic understanding of Node.js and npm (Node Package Manager).

## Requirements

1. **Node.js**: Ensure you have Node.js installed. You can download and install it from [Node.js official website](https://nodejs.org/).
2. **npm**: npm is included with Node.js, so installing Node.js will also install npm.

## Steps to Install @nibiruchain/nibijs

1. **Initialize a New Node.js Project** [OPTIONAL]:
Open your terminal or command prompt and navigate to your project directory. Then, run:

```bash
npm init -y
# yarn init -y
# bun init
```

This will create a package.json file in your project directory.

2. **Install `@nibiruchain/nibijs`**:
Run the following command in your terminal:

```bash
npm install @nibiruchain/nibijs@latest
# yarn add @nibiruchain/nibijs
# bun add @nibiruchain/nibijs
```

3. **Verify Installation:**:
Ensure @nibiruchain/nibijs is listed under dependencies in your package.json file:

```json
{
  "dependencies": {
    "@nibiruchain/nibijs": "^4.4.0"
  }
}
```

This installation process can be used in both node applications and frontend applications, depending on your specific use case.

## Additional Tips

- Updates: Keep your package up-to-date by running the appropriate update command for your package manager:
  - npm: `npm update @nibiruchain/nibijs`
  - yarn: `yarn upgrade @nibiruchain/nibijs`
  - bun: `bun upgrade @nibiruchain/nibijs`

## Troubleshooting

- Common Issues:
  - Ensure you have the correct Node.js version as specified in the documentation.
  - If you encounter permission issues on Unix-based systems, consider using sudo or configure npm to use a different directory for global packages.
  - For Windows, running the command prompt as an administrator may resolve permission issues.
