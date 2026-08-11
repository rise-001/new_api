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

import { parseConsumptionExportDownload } from '../download.ts'

describe('consumption export download responses', () => {
  test('returns the generated workbook and decoded filename', async () => {
    const blob = new Blob(['workbook'])

    const download = await parseConsumptionExportDownload(
      blob,
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      "attachment; filename*=UTF-8''consumption-20260811.xlsx"
    )

    assert.equal(download.blob, blob)
    assert.equal(download.fileName, 'consumption-20260811.xlsx')
  })

  test('surfaces a JSON generation error instead of downloading it', async () => {
    const blob = new Blob([
      JSON.stringify({
        success: false,
        message: 'An export is already being generated',
      }),
    ])

    await assert.rejects(
      parseConsumptionExportDownload(blob, 'application/json; charset=utf-8'),
      { message: 'An export is already being generated' }
    )
  })
})
