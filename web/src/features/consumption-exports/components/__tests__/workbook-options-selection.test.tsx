/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLInputElement',
  'HTMLButtonElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
  'matchMedia',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { CreateExportDialog } = await import('../create-export-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Create one sheet per API key': 'Create one sheet per API key',
        'Export daily summaries without detail rows':
          'Export daily summaries without detail rows',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('consumption export workbook options', () => {
  after(() => {
    domWindow.close()
  })

  test('switches directly between mutually exclusive workbook options', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    queryClient.setQueryData(['api-keys', 'consumption-export-options'], [])
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <CreateExportDialog
              open
              pending={false}
              onOpenChange={() => undefined}
              onCreate={async () => true}
            />
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const groupByToken = document.querySelector('#group-by-token')
    const dailySummary = document.querySelector('#daily-summary')
    assert.ok(groupByToken instanceof HTMLElement)
    assert.ok(dailySummary instanceof HTMLElement)
    assert.equal(groupByToken.hasAttribute('disabled'), false)
    assert.equal(dailySummary.hasAttribute('disabled'), false)

    await act(async () => groupByToken.click())
    assert.equal(groupByToken.getAttribute('aria-checked'), 'true')
    assert.equal(dailySummary.getAttribute('aria-checked'), 'false')

    await act(async () => dailySummary.click())
    assert.equal(groupByToken.getAttribute('aria-checked'), 'false')
    assert.equal(dailySummary.getAttribute('aria-checked'), 'true')
    assert.equal(groupByToken.hasAttribute('disabled'), false)

    await act(async () => groupByToken.click())
    assert.equal(groupByToken.getAttribute('aria-checked'), 'true')
    assert.equal(dailySummary.getAttribute('aria-checked'), 'false')

    await act(async () => root.unmount())
    container.remove()
    queryClient.clear()
  })
})
