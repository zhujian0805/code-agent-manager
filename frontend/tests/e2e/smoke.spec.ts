import { expect, test } from '@playwright/test'

test('desktop frontend smoke navigation', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /^agents$/i })).toBeVisible()
  await page.getByRole('button', { name: /providers/i }).click()
  await expect(page.getByRole('heading', { name: /providers/i })).toBeVisible()
  await page.getByRole('button', { name: /diagnostics/i }).click()
  await expect(page.getByRole('heading', { name: /diagnostics/i })).toBeVisible()
})
