import { api } from '@/lib/api'

export type TransferRecordMode =
  | 'topup'
  | 'add'
  | 'subtract'
  | 'override'
  | 'redemption'

export interface TransferRecord {
  id: number
  user_id: number
  username: string
  created_at: number
  amount: string
  mode: TransferRecordMode | ''
}

interface TransferResponse {
  success: boolean
  data?: { items?: TransferRecord[]; total?: number }
  message?: string
}

export async function getTransferRecords(params: {
  page: number
  pageSize: number
  startTimestamp?: number
  endTimestamp?: number
  username?: string
  mode?: TransferRecordMode | ''
}): Promise<{ items: TransferRecord[]; total: number }> {
  const query = new URLSearchParams({
    p: String(params.page),
    page_size: String(params.pageSize),
  })
  if (params.startTimestamp) {
    query.set('start_timestamp', String(params.startTimestamp))
  }
  if (params.endTimestamp) {
    query.set('end_timestamp', String(params.endTimestamp))
  }
  if (params.username) query.set('username', params.username)
  if (params.mode) query.set('mode', params.mode)
  const res = await api.get<TransferResponse>(`/api/user/transfer?${query}`)
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to load transfer records')
  }
  return { items: res.data.data?.items || [], total: res.data.data?.total || 0 }
}
