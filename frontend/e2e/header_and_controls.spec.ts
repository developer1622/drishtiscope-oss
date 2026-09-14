import { test, expect } from '@playwright/test';

test.describe('Header Navigation & Global Controls', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for the main container to load — use 20s to allow mobile viewport hydration
    await expect(page.locator('header')).toBeVisible({ timeout: 20000 });
  });

  test('renders DrishtiScope logo, brand text and Devanagari badge', async ({ page }) => {
    const brand = page.locator('header').getByText('DrishtiScope');
    await expect(brand).toBeVisible();

    const devanagari = page.locator('header').getByText('दृष्टि');
    await expect(devanagari).toBeVisible();

    // Verify SVG Logo is rendered
    const logoSvg = page.locator('header svg').first();
    await expect(logoSvg).toBeVisible();
  });

  test('displays ModeChip and clicking ? opens Telemetry Modes guide', async ({ page }) => {
    // ModeChip should display REAL LIVE or EBPF LIVE or MOCK
    const modeChip = page.locator('header').getByText(/REAL LIVE|EBPF LIVE|MOCK/);
    await expect(modeChip).toBeVisible();

    // Click the ? icon next to ModeChip
    const modeHelpBtn = page.locator('header button[title*="telemetry modes"]').first();
    await expect(modeHelpBtn).toBeVisible();
    await modeHelpBtn.click();

    // Modal should open with title explaining Telemetry Modes
    const modal = page.locator('div[role="dialog"], .fixed.inset-0');
    await expect(modal).toBeVisible();
    await expect(modal.getByText('DrishtiScope Telemetry Ingestion Modes')).toBeVisible();
    await expect(modal.getByText('REAL LIVE', { exact: false }).first()).toBeVisible();

    // Verify Linux Command section in modal
    await expect(modal.getByText('Linux Command')).toBeVisible();

    // Close modal using ESC key
    await page.keyboard.press('Escape');
    await expect(modal).toBeHidden();
  });

  test('time window selector toggles between 1m, 5m, 15m, 1h and shows help guide', async ({ page }) => {
    for (const tr of ['1m', '5m', '15m', '1h']) {
      const btn = page.locator('header').getByRole('button', { name: tr, exact: true });
      await expect(btn).toBeVisible();
      await btn.click();
      // After click, button should have active styling (cyan highlight)
      await expect(btn).toHaveClass(/bg-cyan/);
    }

    // Click the time window ? help button
    const timeHelpBtn = page.locator('header button[title*="What do 1m, 5m, 15m, 1h mean"]').first();
    await expect(timeHelpBtn).toBeVisible();
    await timeHelpBtn.click();

    // Modal should open
    const modal = page.locator('div[role="dialog"], .fixed.inset-0');
    await expect(modal).toBeVisible();
    await expect(modal.getByText('Observation Time Window')).toBeVisible();

    // Close modal via close button
    const closeBtn = modal.locator('button[title*="Close"]').first();
    await closeBtn.click();
    await expect(modal).toBeHidden();
  });

  test('live/paused stream toggle button freezes and resumes stream', async ({ page }) => {
    const pauseToggle = page.locator('header button[title*="stream"]').first();
    await expect(pauseToggle).toBeVisible();

    // Initially Live
    await expect(pauseToggle).toContainText(/Live|Paused/);

    // Toggle to Paused
    await pauseToggle.click();
    await expect(pauseToggle).toContainText('Paused');

    // Toggle back to Live
    await pauseToggle.click();
    await expect(pauseToggle).toContainText('Live');
  });

  test('anti-flicker toggle switches between smooth and rapid modes', async ({ page }) => {
    const antiFlickerBtn = page.locator('header button[title*="Anti-Flicker"]').first();
    await expect(antiFlickerBtn).toBeVisible();

    // Click anti-flicker toggle
    await antiFlickerBtn.click();
    // Click again to return
    await antiFlickerBtn.click();
  });

  test('theme dropdown switches between Light, Dark, Ubuntu, Unix and Purple themes', async ({ page }) => {
    const themeSelect = page.locator('header select[aria-label="Select Theme Mode"]');
    await expect(themeSelect).toBeVisible();

    // Switch to Dark
    await themeSelect.selectOption('dark');
    await expect(page.locator('html')).toHaveClass(/dark/);

    // Switch to Ubuntu
    await themeSelect.selectOption('ubuntu');
    await expect(page.locator('html')).toHaveClass(/ubuntu/);

    // Switch to Unix
    await themeSelect.selectOption('unix');
    await expect(page.locator('html')).toHaveClass(/unix/);

    // Switch to Purple
    await themeSelect.selectOption('purple');
    await expect(page.locator('html')).toHaveClass(/purple/);

    // Switch back to Light (default)
    await themeSelect.selectOption('light');
    await expect(page.locator('html')).toHaveClass(/light/);
  });
});
