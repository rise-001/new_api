import { createFileRoute } from '@tanstack/react-router'

import { TransferRecords } from '@/features/transfer-records'

export const Route = createFileRoute('/_authenticated/transfer-records/')({
  component: TransferRecords,
})
