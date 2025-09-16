const fs = require('fs');
const fetch = require('node-fetch');
const path = require('path');

const CHAIN_ID = 'cataclysm-1';
const GENESIS_VERSION = 'v1.0.0';
const GENESIS_LINK = `https://github.com/NibiruChain/nibiru/releases/tag/${GENESIS_VERSION}`;
const UPGRADE_URL = `https://networks.nibiru.fi/${CHAIN_ID}/upgrades`;

const TARGET_FILE = path.join(__dirname, '../run-nodes/full-nodes/full-node-mainnet.md');
const INJECT_MARKER_START = '<!-- ::UPGRADES -->';
const INJECT_MARKER_END = '<!-- ::/UPGRADES -->';

function versionFromImage(image) {
  const match = image.match(/nibiru:(.+)$/);
  return match ? match[1] : null;
}

const main = async () => {
  try {
    const res = await fetch(UPGRADE_URL);
    if (!res.ok) throw new Error(`Failed to fetch: ${res.statusText}`);

    const data = await res.json();

    const completedUpgrades = data
      .filter(entry => entry.status === 'completed')
      .sort((a, b) => a.height - b.height);

    const lines = [
      `## Nibiru Mainnet Upgrade Heights`,
      ``,
      `Chain ID: \`${CHAIN_ID}\``,
      ``,
      `Genesis version:   [${GENESIS_VERSION}](${GENESIS_LINK})`,
      ``
    ];

    for (const entry of completedUpgrades) {
      const version = versionFromImage(entry.image);
      if (!version) continue;

      const versionTag = `v${version}`;
      lines.push(`Block \`${entry.height}\`:   [${versionTag}](https://github.com/NibiruChain/nibiru/releases/tag/${versionTag})`);
      lines.push(``);
    }

    const generatedContent = lines.join('\n');

    let targetContent = fs.readFileSync(TARGET_FILE, 'utf8');

    const injectRegex = new RegExp(
      `${INJECT_MARKER_START}[\\s\\S]*?${INJECT_MARKER_END}`,
      'm'
    );

    if (!injectRegex.test(targetContent)) {
      throw new Error(`❌ Could not find inject markers in ${TARGET_FILE}`);
    }

    const newContent = targetContent.replace(
      injectRegex,
      `${INJECT_MARKER_START}\n\n${generatedContent}\n\n${INJECT_MARKER_END}`
    );

    fs.writeFileSync(TARGET_FILE, newContent, 'utf8');
    console.log(`✅ Injected upgrade list into ${TARGET_FILE}`);
  } catch (err) {
    console.error('❌ Error injecting upgrade data:', err);
    process.exit(1);
  }
};

main();