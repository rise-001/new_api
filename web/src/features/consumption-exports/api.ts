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
import { isAxiosError } from 'axios'

import { api } from '@/lib/api'

import { parseConsumptionExportDownload } from './lib/download'
import type {
  ConsumptionExportDownload,
  CreateConsumptionExportInput,
} from './types'

export async function createConsumptionExport(
  input: CreateConsumptionExportInput
): Promise<ConsumptionExportDownload> {
  try {
    const response = await api.post('/api/consumption-export/', input, {
      responseType: 'blob',
      skipBusinessError: true,
      skipErrorHandler: true,
    })
    return parseConsumptionExportDownload(
      response.data as Blob,
      String(response.headers['content-type'] ?? ''),
      response.headers['content-disposition']
    )
  } catch (error) {
    if (isAxiosError(error)) {
      const blob = error.response?.data
      const contentType = String(error.response?.headers['content-type'] ?? '')
      if (blob instanceof Blob && contentType.includes('application/json')) {
        return parseConsumptionExportDownload(blob, contentType)
      }
      throw new Error('Download failed')
    }
    throw error
  }
}
