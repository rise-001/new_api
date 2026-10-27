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
import { after, test } from 'node:test'

import { Window } from 'happy-dom'
import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import zh from '@/i18n/locales/zh.json'

import { DynamicPricingBreakdown } from '../dynamic-pricing-breakdown'

const domWindow = new Window()
after(() => domWindow.close())

const i18n = createInstance()
await i18n.init({ lng: 'zh', resources: { zh } })

const expression =
  '(tier("峰谷价", p * 2 + c * 8)) * (hour("Asia/Shanghai") < 9 ? 0.5 : 1) * (hour("Asia/Shanghai") >= 12 && hour("Asia/Shanghai") < 14 ? 0.5 : 1) * (hour("Asia/Shanghai") >= 18 ? 0.5 : 1)'

function renderRules(options: {
  matched?: number[] | null
  expr?: string
  compact?: boolean
}) {
  const container = domWindow.document.createElement('div')
  container.innerHTML = renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <DynamicPricingBreakdown
        billingExpr={options.expr ?? expression}
        matchedTierLabel='峰谷价'
        matchedMultiplierIndices={options.matched}
        compact={options.compact}
      />
    </I18nextProvider>
  )
  return [...container.querySelectorAll('ul li')]
}

for (const compact of [true, false]) {
  test(`recorded matches highlight only their own multiplier rows (compact=${compact})`, () => {
    const rows = renderRules({ matched: [0, 2], compact })
    assert.equal(rows.length, 3)
    for (const index of [0, 2]) {
      assert.ok(rows[index].textContent.includes('已命中'))
      assert.ok(rows[index].classList.contains('bg-emerald-50/70'))
      assert.ok(rows[index].classList.contains('dark:bg-emerald-500/10'))
      assert.ok(rows[index].textContent.includes('0.5x'))
    }
    assert.ok(!rows[1].textContent.includes('已命中'))
    assert.ok(rows[1].classList.contains('bg-muted/50'))
  })
}

test('old logs and empty match traces do not infer matches from the current time or tier', () => {
  for (const matched of [undefined, null, [], [99]]) {
    const rows = renderRules({ matched })
    assert.equal(rows.length, 3)
    for (const row of rows) {
      assert.ok(!row.textContent.includes('已命中'))
      assert.ok(row.classList.contains('bg-muted/50'))
    }
  }
})

test('zero and unit multipliers show a match when their conditions were true', () => {
  const rows = renderRules({
    matched: [0, 1],
    expr: '(tier("峰谷价", p * 2)) * (hour("UTC") < 9 ? 0 : 1) * (hour("UTC") >= 9 ? 1 : 1)',
  })
  assert.equal(rows.length, 2)
  assert.ok(rows[0].textContent.includes('0x'))
  assert.ok(rows[1].textContent.includes('1x'))
  for (const row of rows) assert.ok(row.textContent.includes('已命中'))
})

test('long matched conditions retain wrapping and both badges in compact details', () => {
  const rows = renderRules({
    compact: true,
    matched: [0],
    expr: '(tier("峰谷价", p * 2)) * (header("x-long-condition-name-for-small-screens") == "long-condition-value-for-small-screens" ? 0.5 : 1)',
  })
  assert.equal(rows.length, 1)
  assert.ok(rows[0].classList.contains('flex-wrap'))
  assert.ok(rows[0].textContent.includes('已命中'))
  assert.ok(rows[0].textContent.includes('0.5x'))
  assert.ok(rows[0].querySelector('span.break-all'))
})
