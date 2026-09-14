import { test, expect } from '@playwright/test';

test.describe('Linux Metric Encyclopedia Modal & Linux Commands', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('header')).toBeVisible({ timeout: 20000 });
  });

  test('clicking ? on KPI opens metric modal with Linux Command cheat sheet', async ({ page }) => {
    // Navigate to Overview tab where KPIs are rendered
    await page.locator('header').getByRole('button', { name: 'Overview' }).click();

    // Click on a metric help button (?)
    const helpBtn = page.locator('main button[title*="What is"]').first();
    await expect(helpBtn).toBeVisible();
    await helpBtn.click();

    // Modal should be visible
    const modal = page.locator('div[role="dialog"], .fixed.inset-0');
    await expect(modal).toBeVisible();

    // Verify sections inside the modal
    await expect(modal.getByText('What Is This? (Simple Terms)')).toBeVisible();
    await expect(modal.getByText('HOW TO UNDERSTAND THE NUMBER')).toBeVisible();
    await expect(modal.getByText('NORMAL / HEALTHY')).toBeVisible();
    await expect(modal.getByText('WARNING / HIGH')).toBeVisible();
    await expect(modal.getByText('CRITICAL / DANGER')).toBeVisible();

    // Verify it says "Linux Command" (and NOT Junior Admin Terminal Command)
    await expect(modal.getByText('Linux Command')).toBeVisible();
    await expect(modal.getByText('Junior Linux Admin Terminal Command')).toBeHidden();

    // Verify Copy Command button is present
    const copyBtn = modal.getByRole('button', { name: /Copy Command/i });
    await expect(copyBtn).toBeVisible();

    // Close modal via ESC
    await page.keyboard.press('Escape');
    await expect(modal).toBeHidden();
  });
});
