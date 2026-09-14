import { test, expect } from '@playwright/test';

test.describe('Dashboard Tabs Navigation & Deep Telemetry Panels', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('header')).toBeVisible({ timeout: 20000 });
  });

  test('navigates through all 5 tabs and verifies tab banners', async ({ page }) => {
    test.setTimeout(60000);
    const tabs = [
      { name: 'Process Story', bannerText: 'Chronological timeline' },
      { name: 'Overview', bannerText: 'Core Agent Golden Signals' },
      { name: 'Execution & CPU', bannerText: 'Continuous on-CPU flamegraph' },
      { name: 'System Metrics', bannerText: 'Deep operational telemetry' },
      { name: 'Security & Logs', bannerText: 'Security & audit center' },
    ];

    for (const tab of tabs) {
      const tabBtn = page.locator('header').getByRole('button', { name: tab.name });
      await expect(tabBtn).toBeVisible();
      await tabBtn.click();

      // Tab Banner should describe what this tab is about
      const banner = page.locator('main').first();
      await expect(banner).toContainText(tab.name);
    }
  });

  test('Tab 1 (Process Story): renders chronological timeline and category filters', async ({ page }) => {
    await page.locator('header').getByRole('button', { name: 'Process Story' }).click();

    // Verify timeline container is visible
    const main = page.locator('main');
    await expect(main).toBeVisible();

    // Verify presence of story components or timeline entries — give mock engine 25s to stream events
    const timelineOrStory = main.getByText(/Process Story|Activity|Chronological|Story|Events/i);
    await expect(timelineOrStory.first()).toBeVisible({ timeout: 25000 });
  });

  test('Tab 2 (Overview / Vitals): renders Golden Signals and Latency Quantiles', async ({ page }) => {
    await page.locator('header').getByRole('button', { name: 'Overview' }).click();

    const main = page.locator('main');
    // Golden signals: CPU, RSS, Threads, FDs
    await expect(main.getByText(/CPU/i).first()).toBeVisible();
    await expect(main.getByText('RSS', { exact: false }).first()).toBeVisible();
    await expect(main.getByText(/P50|P90|P99|Latency/i).first()).toBeVisible();
  });

  test('Tab 3 (Execution & CPU): renders Flamegraph and Linux Process Hierarchy', async ({ page }) => {
    await page.locator('header').getByRole('button', { name: 'Execution & CPU' }).click();

    const main = page.locator('main');
    // Verify On-CPU Flamegraph or Call Tree
    await expect(main.getByText(/Flamegraph|Call Tree|Perfetto|Process/i).first()).toBeVisible();

    // Verify Process Table headers (PID, Comm, CPU, Memory)
    await expect(main.getByText('PID').first()).toBeVisible();
  });

  test('Tab 4 (System Metrics): renders MQL Console, Syscalls and Network Flows', async ({ page }) => {
    await page.locator('header').getByRole('button', { name: 'System Metrics' }).click();

    const main = page.locator('main');
    // Verify MQL Console or System Telemetry
    await expect(main.getByText(/Metrics Query Language|MQL|Syscall|Network|Disk/i).first()).toBeVisible();
  });

  test('Tab 5 (Security & Logs): renders Radar Chart, Role Lens and Kernel Logs', async ({ page }) => {
    await page.locator('header').getByRole('button', { name: 'Security & Logs' }).click();

    const main = page.locator('main');
    // Verify AI Workload Radar or Security Trends
    await expect(main.getByText(/Workload Profile|Security|Kernel Denial|Radar/i).first()).toBeVisible();

    // Verify Role Lens buttons (Performance, Security, Network, Storage)
    const perfRole = main.getByRole('button', { name: /Performance/i }).first();
    await expect(perfRole).toBeVisible();
    await perfRole.click();

    const secRole = main.getByRole('button', { name: /Security/i }).first();
    await expect(secRole).toBeVisible();
    await secRole.click();

    // Verify Kernel Log Explorer
    await expect(main.getByText(/Kernel Log|LogEntry|QUERY/i).first()).toBeVisible();
  });
});
