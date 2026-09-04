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

import { getDefaultTransferRecordDateRange } from '../date-range.ts'

describe('transfer record default date range', () => {
  test('starts at local midnight and ends at the next local midnight', () => {
    const range = getDefaultTransferRecordDateRange(
      new Date(2026, 8, 4, 10, 12, 45)
    )

    assert.deepEqual(range, {
      start: '2026-09-04T00:00',
      end: '2026-09-05T00:00',
    })
  })
})
