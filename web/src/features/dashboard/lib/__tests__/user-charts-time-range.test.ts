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
import { describe, test } from 'node:test'

import { getUserChartsTimeRange } from '../user-charts-time-range'

describe('user analytics time range', () => {
  test('1 day starts at local midnight and ends at the current moment', () => {
    const now = new Date(2026, 9, 27, 15, 30, 45, 123)

    assert.deepEqual(getUserChartsTimeRange(1, now), {
      start_timestamp: new Date(2026, 9, 27).getTime() / 1000,
      end_timestamp: new Date(2026, 9, 27, 15, 30, 45).getTime() / 1000,
    })
    assert.equal(
      now.getTime(),
      new Date(2026, 9, 27, 15, 30, 45, 123).getTime()
    )
  })

  test('1 day at midnight does not include the previous day', () => {
    const now = new Date(2026, 0, 1)

    assert.deepEqual(getUserChartsTimeRange(1, now), {
      start_timestamp: now.getTime() / 1000,
      end_timestamp: now.getTime() / 1000,
    })
  })

  test('longer presets preserve their rolling windows', () => {
    const now = new Date('2026-10-27T15:30:45.000Z')
    const starts = [
      [7, '2026-10-20T15:30:45.000Z'],
      [14, '2026-10-13T15:30:45.000Z'],
      [29, '2026-09-28T15:30:45.000Z'],
    ] as const

    for (const [days, start] of starts) {
      assert.deepEqual(getUserChartsTimeRange(days, now), {
        start_timestamp: new Date(start).getTime() / 1000,
        end_timestamp: now.getTime() / 1000,
      })
    }
  })
})
