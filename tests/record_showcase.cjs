// Playwright Showcase Video Recorder for DrishtiScope
// Records a high-definition 1080p demonstration video of DrishtiScope USP:
// Real-Time Agentic AI & LLM Process Observability, 5 Tabs, 15+ Graphs,
// (?) Metric Guides for Junior Admins, Theme Dropdown, Dynamic Scratchpad, & Copilot.

const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

const VIDEO_DIR = path.join(__dirname, '../videos');
const ARTIFACT_DIR = '/home/ramum/.gemini/antigravity-cli/brain/10549066-662d-4168-81a5-a80f55a3bfde';
const FFMPEG = '/home/ramum/agentscope/frontend/node_modules/ffmpeg-static/ffmpeg';

if (!fs.existsSync(VIDEO_DIR)) {
  fs.mkdirSync(VIDEO_DIR, { recursive: true });
}

async function recordShowcase() {
  console.log('🎥 Starting DrishtiScope Showcase Video Recording...');
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    recordVideo: {
      dir: VIDEO_DIR,
      size: { width: 1920, height: 1080 }
    }
  });

  const page = await context.newPage();

  try {
    console.log('  Opening DrishtiScope in Light Mode (Default)...');
    await page.goto('http://127.0.0.1:5173', { waitUntil: 'networkidle', timeout: 15000 });
    // Let WebSocket snapshot paint
    await page.waitForTimeout(2500);

    // 1. Showcase Header & Status Bar (Default Light Mode)
    console.log('  1. Demonstrating Header & Status Bar in Light Mode...');
    await page.waitForTimeout(1500);

    // 2. Explain Top Controls: Click (?) on Live Stream
    console.log('  2. Demonstrating (?) Help on Live Stream toggle...');
    const liveHelp = await page.locator('button[title*="What does Live / Paused mean"]').first();
    if (await liveHelp.count() > 0) {
      await liveHelp.click();
      await page.waitForTimeout(2200);
      const closeBtn = await page.locator('button[title="Close (ESC)"]').first();
      if (await closeBtn.count() > 0) await closeBtn.click();
      await page.waitForTimeout(800);
    }

    // Explain Top Controls: Click (?) on 1m 5m 15m 1h Time Window
    console.log('  3. Demonstrating (?) Help on Time Window range...');
    const timeHelp = await page.locator('button[title*="What do 1m, 5m, 15m, 1h mean"]').first();
    if (await timeHelp.count() > 0) {
      await timeHelp.click();
      await page.waitForTimeout(2200);
      const closeBtn = await page.locator('button[title="Close (ESC)"]').first();
      if (await closeBtn.count() > 0) await closeBtn.click();
      await page.waitForTimeout(800);
    }

    // 3. Process Story Tab Walkthrough
    console.log('  4. Demonstrating Process Story Tab (Chronological Activity & AI Verdict)...');
    const storyTabBtn = await page.locator('button:has-text("Process Story")').first();
    await storyTabBtn.click();
    await page.waitForTimeout(1500);

    // Smooth scroll down inside main to show files, network tools, and one-click linux diagnostics
    const mainEl = await page.locator('main');
    await mainEl.evaluate((node) => node.scrollTo({ top: 350, behavior: 'smooth' }));
    await page.waitForTimeout(1800);
    await mainEl.evaluate((node) => node.scrollTo({ top: 0, behavior: 'smooth' }));
    await page.waitForTimeout(1000);

    // 4. Overview Tab (Core Agent Golden Signals)
    console.log('  5. Demonstrating Overview Tab (Vitals, Latency Quantiles & Waveform)...');
    const overviewTabBtn = await page.locator('button:has-text("Overview")').first();
    await overviewTabBtn.click();
    await page.waitForTimeout(1500);

    // Click (?) on Latency Quantiles
    const latencyHelp = await page.locator('button[title*="Latency"]').first();
    if (await latencyHelp.count() > 0) {
      await latencyHelp.click();
      await page.waitForTimeout(2000);
      const closeBtn = await page.locator('button[title="Close (ESC)"]').first();
      if (await closeBtn.count() > 0) await closeBtn.click();
      await page.waitForTimeout(800);
    }

    // 5. Execution & CPU Tab (Continuous eBPF Profiler)
    console.log('  6. Demonstrating Execution & CPU Tab (Continuous eBPF Flamegraph & Traces)...');
    const execTabBtn = await page.locator('button:has-text("Execution & CPU")').first();
    await execTabBtn.click();
    await page.waitForTimeout(2500);

    // 6. System Metrics Tab (Deep Telemetry & Donuts)
    console.log('  7. Demonstrating System Metrics Tab (MQL, Subsystem Donuts, IOPS)...');
    const sysTabBtn = await page.locator('button:has-text("System Metrics")').first();
    await sysTabBtn.click();
    await page.waitForTimeout(2500);

    // 7. Security & Logs Tab (Radar & Audit Sandbox)
    console.log('  8. Demonstrating Security & Logs Tab (AI Workload Radar & Denial Trends)...');
    const secTabBtn = await page.locator('button:has-text("Security & Logs")').first();
    await secTabBtn.click();
    await page.waitForTimeout(2500);

    // 8. Theme Tour via Dropdown: Ubuntu -> Unix -> Purple -> Dark -> Light
    console.log('  9. Demonstrating Theme Modes via Dropdown...');
    const themeSelect = await page.locator('select[aria-label="Select Theme Mode"]').first();
    
    // Ubuntu
    await themeSelect.selectOption('ubuntu');
    await page.waitForTimeout(1800);

    // Unix Terminal
    await themeSelect.selectOption('unix');
    await page.waitForTimeout(1800);

    // Cyber Purple
    await themeSelect.selectOption('purple');
    await page.waitForTimeout(1800);

    // Dark
    await themeSelect.selectOption('dark');
    await page.waitForTimeout(1800);

    // Light (Return to default)
    await themeSelect.selectOption('light');
    await page.waitForTimeout(1500);

    // 9. Dynamic Graph Scratchpad Playground
    console.log('  10. Demonstrating Dynamic Graph Scratchpad...');
    // Return to Overview
    await overviewTabBtn.click();
    await page.waitForTimeout(800);
    // Scroll down to scratchpad
    await mainEl.evaluate((node) => node.scrollTo({ top: node.scrollHeight, behavior: 'smooth' }));
    await page.waitForTimeout(1500);

    // Click presets to dynamically add graphs
    const presetThreadMem = await page.locator('button:has-text("+ + Thread vs Memory")').first();
    if (await presetThreadMem.count() > 0) {
      await presetThreadMem.click();
      await page.waitForTimeout(1200);
    }
    const presetPageFaults = await page.locator('button:has-text("+ + Page Faults")').first();
    if (await presetPageFaults.count() > 0) {
      await presetPageFaults.click();
      await page.waitForTimeout(1500);
    }

    // 10. AI Copilot Chat Drawer
    console.log('  11. Demonstrating AI Observability Copilot Chat Drawer...');
    const chatBtn = await page.locator('button[title*="DrishtiScope Agent Copilot"]').first();
    if (await chatBtn.count() > 0) {
      await chatBtn.click();
      await page.waitForTimeout(1200);

      // Click quick prompt
      const quickPrompt = await page.locator('button:has-text("CPU Spike?")').first();
      if (await quickPrompt.count() > 0) {
        await quickPrompt.click();
        await page.waitForTimeout(2500);
      }
    }

    // Final showcase hold
    await page.waitForTimeout(2000);
    console.log('  Showcase walkthrough successfully completed!');

  } catch (err) {
    console.error('Showcase recording error:', err);
  } finally {
    const videoObj = page.video();
    await page.close();
    await context.close();
    await browser.close();

    if (videoObj) {
      const videoPath = await videoObj.path();
      console.log(`\n📹 Raw WebM Video saved at: ${videoPath}`);
      const targetWebm = path.join(VIDEO_DIR, 'drishtiscope_showcase.webm');
      fs.copyFileSync(videoPath, targetWebm);
      fs.copyFileSync(targetWebm, path.join(ARTIFACT_DIR, 'drishtiscope_showcase.webm'));
      console.log(`✅ Saved WebM showcase to: ${targetWebm}`);

      // Transcode to MP4 if ffmpeg binary exists
      if (fs.existsSync(FFMPEG)) {
        const targetMp4 = path.join(VIDEO_DIR, 'drishtiscope_showcase.mp4');
        console.log(`🎬 Transcoding WebM to MP4 using ${FFMPEG}...`);
        try {
          execSync(`"${FFMPEG}" -y -i "${targetWebm}" -c:v libx264 -pix_fmt yuv420p -preset fast -crf 22 "${targetMp4}"`, {
            stdio: 'inherit'
          });
          fs.copyFileSync(targetMp4, path.join(ARTIFACT_DIR, 'drishtiscope_showcase.mp4'));
          console.log(`✅ Transcoded MP4 showcase saved to: ${targetMp4}`);
        } catch (e) {
          console.warn('MP4 transcode failed or skipped:', e.message);
        }
      }
    }
  }
}

recordShowcase().catch((e) => {
  console.error('Fatal error in video recording:', e);
  process.exit(1);
});
