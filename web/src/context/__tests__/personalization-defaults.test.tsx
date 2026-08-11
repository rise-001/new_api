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
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const React = await import('react')
const { act } = React
const { createRoot } = await import('react-dom/client')
const { LayoutProvider, useLayout } = await import('../layout-provider')
const { ThemeCustomizationProvider, useThemeCustomization } =
  await import('../theme-customization-provider')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function ThemeCustomizationProbe() {
  const { customization, defaults } = useThemeCustomization()

  return (
    <output
      data-default-preset={defaults.preset}
      data-preset={customization.preset}
    />
  )
}

function LayoutProbe() {
  const { defaultVariant, variant } = useLayout()

  return <output data-default-variant={defaultVariant} data-variant={variant} />
}

afterEach(() => {
  document.cookie = 'theme_preset=; path=/; max-age=0'
  document.cookie = 'layout_variant=; path=/; max-age=0'
  document.body.removeAttribute('data-theme-preset')
  document.body.removeAttribute('data-theme-font')
})

after(() => {
  domWindow.close()
})

describe('personalization defaults', () => {
  test('applies the Anthropic preset when no theme preference is saved', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <ThemeCustomizationProvider>
          <ThemeCustomizationProbe />
        </ThemeCustomizationProvider>
      )
    })

    const probe = container.querySelector('output')
    assert.ok(probe instanceof HTMLElement)
    assert.equal(probe.dataset.defaultPreset, 'anthropic')
    assert.equal(probe.dataset.preset, 'anthropic')
    assert.equal(document.body.dataset.themePreset, 'anthropic')
    assert.equal(document.body.dataset.themeFont, 'serif')

    await act(async () => root.unmount())
    container.remove()
  })

  test('uses the floating sidebar when no layout preference is saved', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <LayoutProvider>
          <LayoutProbe />
        </LayoutProvider>
      )
    })

    const probe = container.querySelector('output')
    assert.ok(probe instanceof HTMLElement)
    assert.equal(probe.dataset.defaultVariant, 'floating')
    assert.equal(probe.dataset.variant, 'floating')

    await act(async () => root.unmount())
    container.remove()
  })
})
