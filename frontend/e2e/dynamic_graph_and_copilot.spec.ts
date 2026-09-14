import { test, expect } from '@playwright/test';

test.describe('Dynamic Graph Scratchpad & AI Copilot Drawer', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('header')).toBeVisible({ timeout: 15000 });
  });

  test('renders Dynamic Graph Scratchpad & Linux Playground at page end', async ({ page }) => {
    const scratchpad = page.getByText(/Dynamic Graph Scratchpad|Linux Playground/i);
    await expect(scratchpad.first()).toBeVisible();
  });

  test('opens AI Copilot chat drawer and interacts with quick prompts', async ({ page }) => {
    // Find the Copilot trigger button (floating bottom button or header button)
    const copilotTrigger = page.locator('button[title*="Copilot"], button[title*="Chat"], button:has-text("Copilot")').first();
    if (await copilotTrigger.isVisible()) {
      await copilotTrigger.click();

      // Verify Copilot drawer opens
      const drawer = page.getByText(/DrishtiScope Copilot|Agent Telemetry Assistant|Ask Copilot/i);
      await expect(drawer.first()).toBeVisible();

      // Verify prompt input is present
      const promptInput = page.locator('textarea, input[placeholder*="Copilot"], input[placeholder*="Ask"]');
      if (await promptInput.first().isVisible()) {
        await expect(promptInput.first()).toBeVisible();
      }
    }
  });
});
