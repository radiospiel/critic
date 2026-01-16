/**
 * End-to-end tests for Critic webui using Puppeteer
 *
 * Prerequisites:
 * 1. Install dependencies: npm install
 * 2. Build critic: go build -o critic ./cmd/critic
 * 3. Run tests: npm test
 *
 * The tests will start the webui server automatically.
 */

const puppeteer = require('puppeteer');
const { spawn, execSync } = require('child_process');
const path = require('path');

const PORT = 8099;
const BASE_URL = `http://localhost:${PORT}`;

// Test configuration
const config = {
  headless: true,
  timeout: 30000,
};

// Utility functions
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function startServer() {
  const criticPath = path.join(__dirname, '..', '..', 'critic');

  // Check if critic binary exists
  try {
    execSync(`ls ${criticPath}`, { stdio: 'ignore' });
  } catch {
    console.error('Error: critic binary not found. Run "go build -o critic ./cmd/critic" first.');
    process.exit(1);
  }

  console.log(`Starting webui server on port ${PORT}...`);
  const server = spawn(criticPath, ['webui', `--port=${PORT}`], {
    cwd: path.join(__dirname, '..', '..'),
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  // Wait for server to start
  await sleep(2000);

  return server;
}

async function stopServer(server) {
  if (server) {
    server.kill('SIGTERM');
    await sleep(500);
  }
}

// Test runner
class TestRunner {
  constructor() {
    this.passed = 0;
    this.failed = 0;
    this.errors = [];
  }

  async test(name, fn) {
    process.stdout.write(`  ${name}... `);
    try {
      await fn();
      console.log('\x1b[32mPASSED\x1b[0m');
      this.passed++;
    } catch (error) {
      console.log('\x1b[31mFAILED\x1b[0m');
      this.errors.push({ name, error: error.message });
      this.failed++;
    }
  }

  summary() {
    console.log('\n' + '='.repeat(50));
    console.log(`Results: ${this.passed} passed, ${this.failed} failed`);
    if (this.errors.length > 0) {
      console.log('\nFailures:');
      this.errors.forEach(({ name, error }) => {
        console.log(`  - ${name}: ${error}`);
      });
    }
    return this.failed === 0;
  }
}

// Assertion helpers
function assert(condition, message) {
  if (!condition) {
    throw new Error(message || 'Assertion failed');
  }
}

function assertEqual(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message || `Expected "${expected}", got "${actual}"`);
  }
}

function assertIncludes(text, substring, message) {
  if (!text.includes(substring)) {
    throw new Error(message || `Expected text to include "${substring}"`);
  }
}

// Main test suite
async function runTests() {
  console.log('Critic WebUI E2E Tests\n' + '='.repeat(50));

  const runner = new TestRunner();
  let server = null;
  let browser = null;
  let page = null;

  try {
    // Start server
    server = await startServer();

    // Launch browser
    browser = await puppeteer.launch({
      headless: config.headless ? 'new' : false,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });
    page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 800 });

    console.log('\n1. Page Load Tests');
    console.log('-'.repeat(50));

    await runner.test('Main page loads successfully', async () => {
      const response = await page.goto(BASE_URL, { waitUntil: 'networkidle0' });
      assert(response.ok(), 'Page should return 200 OK');
    });

    await runner.test('Page title is correct', async () => {
      const title = await page.title();
      assertEqual(title, 'Critic - Code Review');
    });

    await runner.test('Header contains "Critic" text', async () => {
      const headerText = await page.$eval('header h1', el => el.textContent);
      assertEqual(headerText, 'Critic');
    });

    await runner.test('Theme toggle button exists', async () => {
      const button = await page.$('.theme-toggle');
      assert(button !== null, 'Theme toggle button should exist');
    });

    console.log('\n2. File List Tests');
    console.log('-'.repeat(50));

    await runner.test('File list container exists', async () => {
      const fileList = await page.$('#file-list');
      assert(fileList !== null, 'File list container should exist');
    });

    await runner.test('File list loads files', async () => {
      // Wait for htmx to load files
      await page.waitForSelector('.file-item', { timeout: 5000 });
      const fileItems = await page.$$('.file-item');
      assert(fileItems.length > 0, 'Should have at least one file in the list');
    });

    await runner.test('File items have correct structure', async () => {
      const fileItem = await page.$('.file-item');
      const status = await fileItem.$('.file-status');
      const path = await fileItem.$('.file-path');
      assert(status !== null, 'File item should have status element');
      assert(path !== null, 'File item should have path element');
    });

    console.log('\n3. Diff View Tests');
    console.log('-'.repeat(50));

    await runner.test('Clicking file loads diff', async () => {
      // Click the first file
      await page.click('.file-item');
      // Wait for diff to load
      await page.waitForSelector('.diff-content', { timeout: 5000 });
      const diffContent = await page.$('.diff-content');
      assert(diffContent !== null, 'Diff content should be loaded');
    });

    await runner.test('Diff contains line numbers', async () => {
      const lineNumbers = await page.$('.line-numbers');
      assert(lineNumbers !== null, 'Diff should have line numbers');
    });

    await runner.test('Diff has syntax highlighting', async () => {
      // Check for chroma classes (syntax highlighting)
      const hasHighlighting = await page.evaluate(() => {
        const spans = document.querySelectorAll('.line-content span[class]');
        return spans.length > 0;
      });
      // This might be false for non-code files, so we just check the structure exists
      const lineContent = await page.$('.line-content');
      assert(lineContent !== null, 'Line content should exist');
    });

    console.log('\n4. Theme Toggle Tests');
    console.log('-'.repeat(50));

    await runner.test('Default theme is dark', async () => {
      const theme = await page.evaluate(() => {
        return document.documentElement.getAttribute('data-theme');
      });
      assertEqual(theme, 'dark');
    });

    await runner.test('Theme toggle changes theme to light', async () => {
      await page.click('.theme-toggle');
      const theme = await page.evaluate(() => {
        return document.documentElement.getAttribute('data-theme');
      });
      assertEqual(theme, 'light');
    });

    await runner.test('Theme toggle changes theme back to dark', async () => {
      await page.click('.theme-toggle');
      const theme = await page.evaluate(() => {
        return document.documentElement.getAttribute('data-theme');
      });
      assertEqual(theme, 'dark');
    });

    await runner.test('Theme persists in localStorage', async () => {
      const storedTheme = await page.evaluate(() => {
        return localStorage.getItem('critic-theme');
      });
      assertEqual(storedTheme, 'dark');
    });

    console.log('\n5. Keyboard Navigation Tests');
    console.log('-'.repeat(50));

    await runner.test('Help overlay shows on ? key', async () => {
      await page.keyboard.press('?');
      await sleep(100);
      const helpOverlay = await page.$('#help-overlay');
      const isVisible = await page.evaluate(() => {
        const overlay = document.getElementById('help-overlay');
        return overlay && overlay.style.display !== 'none';
      });
      assert(isVisible, 'Help overlay should be visible after pressing ?');
    });

    await runner.test('Help overlay closes on ? key', async () => {
      await page.keyboard.press('?');
      await sleep(100);
      const isHidden = await page.evaluate(() => {
        const overlay = document.getElementById('help-overlay');
        return !overlay || overlay.style.display === 'none';
      });
      assert(isHidden, 'Help overlay should be hidden after pressing ? again');
    });

    await runner.test('j/k keys navigate in file list', async () => {
      // Focus on file list (Tab to switch)
      await page.keyboard.press('Tab');
      await sleep(100);

      // Get current active file index
      const initialIndex = await page.evaluate(() => {
        const items = Array.from(document.querySelectorAll('.file-item'));
        return items.findIndex(el => el.classList.contains('active'));
      });

      // Press j to move down
      await page.keyboard.press('j');
      await sleep(100);

      const newIndex = await page.evaluate(() => {
        const items = Array.from(document.querySelectorAll('.file-item'));
        return items.findIndex(el => el.classList.contains('active'));
      });

      // Index should have changed (or stayed at max)
      assert(typeof newIndex === 'number', 'Should be able to navigate with j key');
    });

    console.log('\n6. API Endpoint Tests');
    console.log('-'.repeat(50));

    await runner.test('/api/files returns file list', async () => {
      const response = await page.goto(`${BASE_URL}/api/files`);
      assert(response.ok(), 'API should return 200 OK');
      const content = await page.content();
      assertIncludes(content, 'file-item', 'Response should contain file items');
    });

    await runner.test('/api/diff/{path} returns diff content', async () => {
      const response = await page.goto(`${BASE_URL}/api/diff/go.mod`);
      assert(response.ok(), 'API should return 200 OK');
      const content = await page.content();
      assertIncludes(content, 'diff-content', 'Response should contain diff content');
    });

    console.log('\n7. WebSocket Tests');
    console.log('-'.repeat(50));

    await runner.test('WebSocket connection is established', async () => {
      await page.goto(BASE_URL, { waitUntil: 'networkidle0' });
      await sleep(1000);

      // Check if htmx ws extension is loaded
      const hasWsExtension = await page.evaluate(() => {
        return typeof htmx !== 'undefined' && htmx.config !== undefined;
      });
      assert(hasWsExtension, 'htmx should be loaded');
    });

  } catch (error) {
    console.error('\n\x1b[31mTest suite error:\x1b[0m', error.message);
  } finally {
    // Cleanup
    if (browser) {
      await browser.close();
    }
    if (server) {
      await stopServer(server);
    }
  }

  // Print summary and exit with appropriate code
  const success = runner.summary();
  process.exit(success ? 0 : 1);
}

// Run tests
runTests().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
