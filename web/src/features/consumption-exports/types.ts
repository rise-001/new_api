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
export type ConsumptionExportStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'canceled'

export type ConsumptionExportStatusFilter = ConsumptionExportStatus | 'all'

export type ConsumptionExportPayload = {
  start_timestamp: number
  end_timestamp: number
  token_id: number
  token_name: string
  group_by_token: boolean
  daily_summary: boolean
  timezone_offset: number
}

export type ConsumptionExportState = {
  total: number
  processed: number
  progress: number
  stage: string
}

export type ConsumptionExportResult = {
  record_count: number
  file_name: string
  file_size: number
  expires_at: number
  model_count: number
}

export type ConsumptionExportTask = {
  id: number
  task_id: string
  status: ConsumptionExportStatus
  payload: ConsumptionExportPayload
  state: ConsumptionExportState
  result: ConsumptionExportResult
  error: string
  available: boolean
  created_at: number
  updated_at: number
}

export type ConsumptionExportPage = {
  page: number
  page_size: number
  total: number
  items: ConsumptionExportTask[]
}

export type CreateConsumptionExportInput = {
  start_timestamp: number
  end_timestamp: number
  token_id: number
  group_by_token: boolean
  daily_summary: boolean
  timezone_offset: number
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
