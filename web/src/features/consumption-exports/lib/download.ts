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
import type { ConsumptionExportDownload } from '../types'

export async function parseConsumptionExportDownload(
  blob: Blob,
  contentType: string,
  contentDisposition?: string
): Promise<ConsumptionExportDownload> {
  if (contentType.includes('application/json')) {
    let message = 'Download failed'
    try {
      const response = JSON.parse(await blob.text()) as { message?: unknown }
      if (typeof response.message === 'string' && response.message) {
        message = response.message
      }
    } catch {
      // Keep the stable fallback for malformed error responses.
    }
    throw new Error(message)
  }

  let fileName = 'consumption.xlsx'
  const encodedFileName = /filename\*=UTF-8''([^;]+)/i.exec(
    contentDisposition ?? ''
  )?.[1]
  if (encodedFileName) {
    try {
      fileName = decodeURIComponent(encodedFileName)
    } catch {
      fileName = 'consumption.xlsx'
    }
  }
  return { blob, fileName }
}
