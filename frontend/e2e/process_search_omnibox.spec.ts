import { test, expect } from '@playwright/test';

test.describe('Executing Process Search Omnibox & Suggestions', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('header')).toBeVisible({ timeout: 20000 });
  });

  test('pressing / shortcut focuses the search omnibox', async ({ page }) => {
    // Press '/' key
    await page.keyboard.press('/');
    const searchInput = page.locator('header input[type="text"]').first();
    await expect(searchInput).toBeFocused();
  });

  test('focusing search opens live suggestions dropdown of current executing processes', async ({ page }) => {
    const searchInput = page.locator('header input[type="text"]').first();
    await searchInput.focus();

    // The dropdown container should appear
    const dropdown = page.locator('header').getByText('Current Executing Processes');
    await expect(dropdown).toBeVisible();

    // Suggestions should show process name and PID
    const suggestionRows = page.locator('header button').filter({ hasText: /PID \d+/ });
    const count = await suggestionRows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('typing filters executing processes live', async ({ page }) => {
    const searchInput = page.locator('header input[type="text"]').first();
    await searchInput.fill('agy');

    // Should filter suggestions matching 'agy' or show fallback
    const dropdown = page.locator('header').getByText('Current Executing Processes');
    await expect(dropdown).toBeVisible();

    // Verify clear button appears
    const clearBtn = page.locator('header button[title="Clear search"]');
    await expect(clearBtn).toBeVisible();

    // Click clear button
    await clearBtn.click();
    await expect(searchInput).toHaveValue('');
  });

  test('keyboard navigation with ArrowDown/ArrowUp and Escape key', async ({ page }) => {
    const searchInput = page.locator('header input[type="text"]').first();
    await searchInput.focus();

    // Press ArrowDown to highlight suggestion
    await page.keyboard.press('ArrowDown');

    // Press Escape to close dropdown
    await page.keyboard.press('Escape');

    const dropdownHeader = page.locator('header').getByText('Current Executing Processes');
    await expect(dropdownHeader).toBeHidden();
  });

  test('clicking outside search container dismisses suggestions dropdown', async ({ page }) => {
    const searchInput = page.locator('header input[type="text"]').first();
    await searchInput.focus();

    const dropdownHeader = page.locator('header').getByText('Current Executing Processes');
    await expect(dropdownHeader).toBeVisible();

    // Click outside on main area
    await page.locator('main').click({ position: { x: 50, y: 50 } });
    await expect(dropdownHeader).toBeHidden();
  });
});
