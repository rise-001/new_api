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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  ConsumptionExportPage,
  ConsumptionExportStatusFilter,
  CreateConsumptionExportInput,
} from './types'

export async function listConsumptionExports(params: {
  page: number
  pageSize: number
  status: ConsumptionExportStatusFilter
}): Promise<ApiResponse<ConsumptionExportPage>> {
  const query = new URLSearchParams({
    p: String(params.page),
    size: String(params.pageSize),
  })
  if (params.status !== 'all') query.set('status', params.status)
  const response = await api.get(`/api/consumption-export/?${query}`)
  return response.data
}

export async function createConsumptionExport(
  input: CreateConsumptionExportInput
): Promise<ApiResponse<{ task_id: string; status: string; created: boolean }>> {
  const response = await api.post('/api/consumption-export/', input)
  return response.data
}

export async function cancelConsumptionExport(
  taskId: string
): Promise<ApiResponse<null>> {
  const response = await api.post(
    `/api/consumption-export/${encodeURIComponent(taskId)}/cancel`
  )
  return response.data
}

export async function deleteConsumptionExport(
  taskId: string
): Promise<ApiResponse<null>> {
  const response = await api.delete(
    `/api/consumption-export/${encodeURIComponent(taskId)}`
  )
  return response.data
}

export async function downloadConsumptionExport(taskId: string): Promise<Blob> {
  const response = await api.get(
    `/api/consumption-export/${encodeURIComponent(taskId)}/download`,
    { responseType: 'blob', disableDuplicate: true }
  )
  return response.data as Blob
}
