import { homedir } from "os";
import { join } from "path";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";

// Read version from package.json at build time
const pkg = require("../package.json");
export const VERSION = pkg.version;

const CACHE_DIR = join(homedir(), ".tdx");
const CACHE_FILE = join(CACHE_DIR, "version-cache.json");
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000; // 24 hours
const GITHUB_API = "https://api.github.com/repos/niklas-heer/tdx/releases/latest";

interface VersionCache {
  latestVersion: string;
  checkedAt: number;
}

function ensureCacheDir(): void {
  if (!existsSync(CACHE_DIR)) {
    mkdirSync(CACHE_DIR, { recursive: true });
  }
}

function readCache(): VersionCache | null {
  try {
    if (existsSync(CACHE_FILE)) {
      const data = readFileSync(CACHE_FILE, "utf-8");
      return JSON.parse(data);
    }
  } catch {
    // Ignore cache read errors
  }
  return null;
}

function writeCache(cache: VersionCache): void {
  try {
    ensureCacheDir();
    writeFileSync(CACHE_FILE, JSON.stringify(cache));
  } catch {
    // Ignore cache write errors
  }
}

async function fetchLatestVersion(): Promise<string | null> {
  try {
    const response = await fetch(GITHUB_API, {
      headers: {
        "Accept": "application/vnd.github.v3+json",
        "User-Agent": "tdx-update-checker"
      }
    });

    if (!response.ok) return null;

    const data = await response.json() as { tag_name: string };
    // Remove 'v' prefix if present
    return data.tag_name.replace(/^v/, "");
  } catch {
    return null;
  }
}

function compareVersions(current: string, latest: string): number {
  const currentParts = current.split(".").map(Number);
  const latestParts = latest.split(".").map(Number);

  for (let i = 0; i < 3; i++) {
    const c = currentParts[i] || 0;
    const l = latestParts[i] || 0;
    if (l > c) return 1;  // Latest is newer
    if (l < c) return -1; // Current is newer
  }
  return 0; // Same version
}

export async function checkForUpdates(): Promise<void> {
  // Skip if disabled
  if (process.env.TDX_NO_UPDATE_CHECK === "1") {
    return;
  }

  const cache = readCache();
  const now = Date.now();

  // Check if we need to fetch new version
  if (cache && (now - cache.checkedAt) < CHECK_INTERVAL_MS) {
    // Use cached version
    if (compareVersions(VERSION, cache.latestVersion) > 0) {
      showUpdateMessage(cache.latestVersion);
    }
    return;
  }

  // Fetch latest version
  const latestVersion = await fetchLatestVersion();

  if (latestVersion) {
    // Update cache
    writeCache({
      latestVersion,
      checkedAt: now
    });

    // Show update message if newer version available
    if (compareVersions(VERSION, latestVersion) > 0) {
      showUpdateMessage(latestVersion);
    }
  }
}

function showUpdateMessage(latestVersion: string): void {
  console.log(`\x1b[33m⚠ Update available: ${VERSION} → ${latestVersion}\x1b[0m`);
  console.log(`\x1b[2m  Run: brew upgrade tdx  or  curl -fsSL https://raw.githubusercontent.com/niklas-heer/tdx/main/scripts/install.sh | bash\x1b[0m`);
  console.log("");
}

export function showVersion(): void {
  console.log(`tdx version ${VERSION}`);
}
